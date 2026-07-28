package container

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"graft/server/internal/moduleapi"
)

const (
	updateComposeCandidateConfidenceHigh   = "high"
	updateComposeCandidateConfidenceMedium = "medium"
)

// DiscoverCurrentServerCompose 通过当前进程容器的 hostname inspect Docker 元数据，生成一次性 Compose 根目录候选。
// 它不读取宿主机文件，也不把 Docker inspect 原始载荷暴露给 Update；候选根目录必须来自 Compose 元数据或 bind source。
//
//nolint:cyclop,gocognit,gocyclo // 候选归并、来源置信度和稳定排序共同构成一次发现快照。
func (r containerProjectRuntimeReader) DiscoverCurrentServerCompose(ctx context.Context) ([]moduleapi.UpdateComposeRuntimeCandidate, error) {
	if r.service == nil {
		return nil, errRuntimeDisabled
	}
	runtime, err := r.service.runtimeForRequestContext(ctx)
	if err != nil {
		return nil, err
	}
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return nil, errors.New("current server container identity is unavailable")
	}
	detail, err := runtime.Detail(ctx, Ref{Value: strings.TrimSpace(hostname)})
	if err != nil {
		return nil, err
	}
	labels := detail.Labels
	workingDir := strings.TrimSpace(labels[composeWorkingDirLabel])
	configFiles := composeConfigFiles(labels[composeConfigFilesLabel])
	if workingDir == "" && len(configFiles) > 0 && filepath.IsAbs(configFiles[0]) {
		workingDir = filepath.Dir(configFiles[0])
	}

	values := make(map[string]moduleapi.UpdateComposeRuntimeCandidate)
	add := func(root, confidence string, warnings []string) {
		root = filepath.Clean(strings.TrimSpace(root))
		if !filepath.IsAbs(root) || root == string(filepath.Separator) {
			return
		}
		files := make([]string, 0, len(configFiles))
		for _, file := range configFiles {
			file = filepath.Clean(file)
			if filepath.IsAbs(file) && pathWithinRoot(root, file) {
				files = append(files, file)
			}
		}
		if len(files) == 0 && confidence == updateComposeCandidateConfidenceHigh {
			files = []string{filepath.Join(root, "compose.yml")}
		}
		keyInput := strings.Join([]string{root, strings.Join(files, "\x00")}, "\x00")
		digest := sha256.Sum256([]byte(keyInput))
		key := "compose-" + hex.EncodeToString(digest[:8])
		candidate := moduleapi.UpdateComposeRuntimeCandidate{CandidateKey: key, Root: root, WorkingDir: root, ConfigFiles: files, ProjectName: strings.TrimSpace(labels[composeProjectLabel]), Confidence: confidence, Warnings: append([]string(nil), warnings...)}
		if existing, ok := values[key]; ok && existing.Confidence == updateComposeCandidateConfidenceHigh {
			return
		}
		values[key] = candidate
	}
	if workingDir != "" {
		add(workingDir, updateComposeCandidateConfidenceHigh, nil)
	}
	// Compose 标签同时标识根目录与文件序列；仅在该高置信事实缺失时才回退 bind mount，避免调用方在无关数据挂载中选择。
	if len(values) == 0 {
		for _, mount := range detail.Mounts {
			if mount.Type != "bind" || !filepath.IsAbs(strings.TrimSpace(mount.Source)) {
				continue
			}
			warnings := []string{"bind_mount_candidate_requires_administrator_confirmation"}
			add(mount.Source, updateComposeCandidateConfidenceMedium, warnings)
		}
	}
	candidates := make([]moduleapi.UpdateComposeRuntimeCandidate, 0, len(values))
	for _, candidate := range values {
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Confidence != candidates[j].Confidence {
			return candidates[i].Confidence == updateComposeCandidateConfidenceHigh
		}
		return candidates[i].Root < candidates[j].Root
	})
	return candidates, nil
}

func pathWithinRoot(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." {
		return false
	}
	return !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != "."
}

var _ moduleapi.UpdateComposeRuntimeReader = containerProjectRuntimeReader{}
