package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
)

const maxApplicationWorkspaceSnapshotBytes int64 = 512 << 20
const applicationWorkspaceDirectoryMode = 0o750

// ResolveApplicationBuildContext 在不暴露 Project 持久化模型的前提下返回 Build 所需的已授权工作区事实。
//
//nolint:cyclop // 授权、存活应用和可用 Docker target 必须在同一资源边界内逐项确认。
func (s *Service) ResolveApplicationBuildContext(ctx context.Context, applicationID string) (moduleapi.ApplicationBuildContext, error) {
	if !isApplicationID(applicationID) || s == nil || s.authorizer == nil {
		return moduleapi.ApplicationBuildContext{}, errProjectInvalidArgument
	}
	auth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || auth.User == nil {
		return moduleapi.ApplicationBuildContext{}, moduleapi.ErrUnauthenticated
	}
	if err := s.authorizer.Authorize(ctx, auth, projectcontract.ApplicationViewPermission.String()); err != nil {
		return moduleapi.ApplicationBuildContext{}, err
	}
	applicationRecordID, err := s.ResolveApplicationID(ctx, applicationID)
	if err != nil {
		return moduleapi.ApplicationBuildContext{}, err
	}
	aggregate, err := s.getAggregate(ctx, applicationRecordID)
	if err != nil {
		return moduleapi.ApplicationBuildContext{}, err
	}
	item := aggregate.Application
	if item.RuntimeTargetID == nil || item.WorkspacePath == "" {
		return moduleapi.ApplicationBuildContext{}, errors.New("application build context is unavailable")
	}
	if *item.RuntimeTargetID > math.MaxInt64 {
		return moduleapi.ApplicationBuildContext{}, errProjectInvalidArgument
	}
	targetID := int64(*item.RuntimeTargetID) // #nosec G115 -- 已在转换前限制为 PostgreSQL bigint 可表示范围。
	target, err := s.runtimeTargets.ReadComposeTarget(ctx, &targetID)
	if err != nil {
		return moduleapi.ApplicationBuildContext{}, err
	}
	if target.Provider != "docker" {
		return moduleapi.ApplicationBuildContext{}, errors.New("application runtime target does not support Docker builds")
	}
	return moduleapi.ApplicationBuildContext{
		ApplicationID: item.ApplicationID, ApplicationRecordID: item.ApplicationRecordID, DisplayName: item.DisplayName, WorkspaceRoot: item.WorkspacePath,
		RuntimeTargetID: *item.RuntimeTargetID, RuntimeTargetName: target.DisplayName, RuntimeProvider: target.Provider, CanBuild: true,
	}, nil
}

// FreezeApplicationWorkspaceSnapshot 将已授权 Application workspace 捕获为
// content-addressed Build input；executor-local root 只经由 moduleapi 返回，
// 绝不由 HTTP 暴露。
func (s *Service) FreezeApplicationWorkspaceSnapshot(ctx context.Context, applicationID string) (moduleapi.WorkspaceSnapshot, moduleapi.ApplicationBuildContext, error) {
	buildContext, err := s.ResolveApplicationBuildContext(ctx, applicationID)
	if err != nil {
		return moduleapi.WorkspaceSnapshot{}, moduleapi.ApplicationBuildContext{}, err
	}
	if !filepath.IsAbs(buildContext.WorkspaceRoot) {
		return moduleapi.WorkspaceSnapshot{}, moduleapi.ApplicationBuildContext{}, errors.New("application workspace root is invalid")
	}
	s.workspaceMutationMu.Lock()
	defer s.workspaceMutationMu.Unlock()
	digest, err := applicationWorkspaceDigest(buildContext.WorkspaceRoot)
	if err != nil {
		return moduleapi.WorkspaceSnapshot{}, moduleapi.ApplicationBuildContext{}, err
	}
	materializedRoot, err := materializeApplicationWorkspace(buildContext.WorkspaceRoot)
	if err != nil {
		return moduleapi.WorkspaceSnapshot{}, moduleapi.ApplicationBuildContext{}, err
	}
	materializedDigest, err := applicationWorkspaceDigest(materializedRoot)
	if err != nil || materializedDigest != digest {
		_ = os.RemoveAll(materializedRoot)
		if err != nil {
			return moduleapi.WorkspaceSnapshot{}, moduleapi.ApplicationBuildContext{}, err
		}
		return moduleapi.WorkspaceSnapshot{}, moduleapi.ApplicationBuildContext{}, errors.New("application workspace changed while snapshot was materialized")
	}
	return moduleapi.WorkspaceSnapshot{
		ID:               "application:" + buildContext.ApplicationID + ":" + digest[:24],
		SourceKind:       "application_workspace",
		SourceReference:  buildContext.ApplicationID,
		ContentDigest:    "sha256:" + digest,
		MaterializedRoot: materializedRoot,
		CreatedAt:        time.Now().UTC(),
	}, buildContext, nil
}

// materializeApplicationWorkspace 只将 regular file 与 directory 复制到私有
// temporary root；即使 Application workspace 后续变化，该副本也使 retry 消费同一 source。
//
//nolint:gocognit,cyclop // copy boundary 刻意校验每一个 filesystem entry。
func materializeApplicationWorkspace(root string) (string, error) {
	destination, err := os.MkdirTemp("", "graft-build-snapshot-")
	if err != nil {
		return "", err
	}
	failed := true
	defer func() {
		if failed {
			_ = os.RemoveAll(destination)
		}
	}()
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("application workspace contains symbolic link")
		}
		output := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(output, applicationWorkspaceDirectoryMode)
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			if err != nil {
				return err
			}
			return errors.New("application workspace contains unsupported entry")
		}
		source, err := os.Open(path) // #nosec G122,G304 -- path is enumerated from the authorized Application workspace.
		if err != nil {
			return err
		}
		defer func() { _ = source.Close() }()
		outputFile, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm()) // #nosec G304 -- output is derived below the private snapshot root.
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(outputFile, source)
		closeErr := outputFile.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		return "", err
	}
	failed = false
	return destination, nil
}

//nolint:gocognit,cyclop // snapshot digest 必须确定性地 inspect 并分类每一个 entry。
func applicationWorkspaceDigest(root string) (string, error) {
	entries := make([]string, 0)
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == "." {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("application workspace contains symbolic link")
		}
		if entry.IsDir() {
			entries = append(entries, "d:"+filepath.ToSlash(relative))
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			if err != nil {
				return err
			}
			return errors.New("application workspace contains unsupported entry")
		}
		total += info.Size()
		if total > maxApplicationWorkspaceSnapshotBytes {
			return errors.New("application workspace snapshot exceeds size limit")
		}
		content, err := os.ReadFile(path) // #nosec G122,G304 -- path is enumerated from the authorized Application workspace.
		if err != nil {
			return err
		}
		fileHash := sha256.Sum256(content)
		entries = append(entries, "f:"+filepath.ToSlash(relative)+":"+hex.EncodeToString(fileHash[:]))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(entries)
	hash := sha256.New()
	for _, entry := range entries {
		if _, err := io.WriteString(hash, entry+"\n"); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
