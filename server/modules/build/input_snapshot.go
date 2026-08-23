package build

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

const (
	maxInputSnapshotUploadBytes   int64 = 100 << 20
	maxInputSnapshotExpandedBytes int64 = 1 << 30
	maxInputSnapshotEntries             = 10000
	archiveFileMode                     = os.FileMode(0o777)
)

// InputSnapshotUpload 是 Build-owned 的一次性归档上传输入。
type InputSnapshotUpload struct {
	Archive io.Reader
	Size    int64
	UserID  uint64
}

// CreateInputSnapshot 安全解包并冻结上传内容；响应只包含 Snapshot 身份摘要，
// 物化路径始终留在 Build 内部。
//
//nolint:gocognit,gocyclo,cyclop // 上传边界集中执行大小、格式、路径、摘要和生命周期校验。
func (s *Service) CreateInputSnapshot(ctx context.Context, upload InputSnapshotUpload) (moduleapi.WorkspaceSnapshot, error) {
	if s == nil || s.repository == nil || upload.Archive == nil || upload.Size < 0 || upload.Size > maxInputSnapshotUploadBytes {
		return moduleapi.WorkspaceSnapshot{}, errInvalidBuildRequest
	}
	repository, ok := s.repository.(buildstore.InputSnapshotRepository)
	if !ok {
		return moduleapi.WorkspaceSnapshot{}, errors.New("build input snapshot persistence is unavailable")
	}
	content, err := readBoundedArchive(upload.Archive, upload.Size)
	if err != nil {
		return moduleapi.WorkspaceSnapshot{}, err
	}
	root, err := extractInputArchive(content)
	if err != nil {
		return moduleapi.WorkspaceSnapshot{}, err
	}
	digest, err := materializationDigest(root)
	if err != nil {
		_ = os.RemoveAll(root)
		return moduleapi.WorkspaceSnapshot{}, fmt.Errorf("digest input snapshot: %w", err)
	}
	if _, err := os.Stat(filepath.Join(root, "Dockerfile")); err != nil {
		_ = os.RemoveAll(root)
		return moduleapi.WorkspaceSnapshot{}, errors.New("input snapshot must contain a root Dockerfile")
	}
	snapshotID := newSnapshotIdentity()
	reference, err := moduleapi.NewWorkspaceSnapshotMaterializationReference(snapshotID, digest, root)
	if err != nil {
		_ = os.RemoveAll(root)
		return moduleapi.WorkspaceSnapshot{}, err
	}
	snapshot := moduleapi.WorkspaceSnapshot{ID: snapshotID, SourceKind: moduleapi.WorkspaceSourceArchive, SourceReference: "upload:" + digest, ContentDigest: digest, MaterializationRef: reference, CreatedAt: time.Now().UTC()}
	created, err := repository.CreateBuildInputSnapshot(ctx, snapshot, upload.UserID)
	if err != nil {
		_ = os.RemoveAll(root)
		return moduleapi.WorkspaceSnapshot{}, err
	}
	if created.MaterializationRef != snapshot.MaterializationRef {
		_ = releaseMaterialization(snapshot.MaterializationRef)
	}
	return created, nil
}

func readBoundedArchive(reader io.Reader, declaredSize int64) ([]byte, error) {
	if declaredSize > maxInputSnapshotUploadBytes {
		return nil, errors.New("input snapshot archive exceeds upload limit")
	}
	limited := io.LimitReader(reader, maxInputSnapshotUploadBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read input snapshot archive: %w", err)
	}
	if int64(len(data)) > maxInputSnapshotUploadBytes {
		return nil, errors.New("input snapshot archive exceeds upload limit")
	}
	return data, nil
}

//nolint:gocognit,gocyclo,cyclop,nestif // 归档格式探测和受控临时目录初始化必须在同一安全边界。
func extractInputArchive(data []byte) (string, error) {
	root, err := os.MkdirTemp(filepath.Join(os.TempDir(), "graft-build-snapshots"), "snapshot-")
	if err != nil {
		if os.IsNotExist(err) {
			if mkdirErr := os.MkdirAll(filepath.Join(os.TempDir(), "graft-build-snapshots"), managedSnapshotDirectoryMode); mkdirErr != nil {
				return "", mkdirErr
			}
			root, err = os.MkdirTemp(filepath.Join(os.TempDir(), "graft-build-snapshots"), "snapshot-")
		}
		if err != nil {
			return "", err
		}
	}
	failed := true
	defer func() {
		if failed {
			_ = os.RemoveAll(root)
		}
	}()
	if len(data) >= 2 && data[0] == 'P' && data[1] == 'K' {
		err = extractZip(root, data)
	} else {
		payload := io.Reader(bytes.NewReader(data))
		if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
			gzipReader, gzipErr := gzip.NewReader(bytes.NewReader(data))
			if gzipErr != nil {
				return "", errors.New("invalid TAR.GZ archive")
			}
			decompressed, readErr := io.ReadAll(io.LimitReader(gzipReader, maxInputSnapshotExpandedBytes+1))
			_ = gzipReader.Close()
			if readErr != nil || int64(len(decompressed)) > maxInputSnapshotExpandedBytes {
				return "", errors.New("input snapshot archive exceeds extracted size limit")
			}
			payload = bytes.NewReader(decompressed)
		}
		err = extractTarReader(root, payload)
	}
	if err != nil {
		return "", err
	}
	failed = false
	return root, nil
}

//nolint:gocognit,gocyclo,cyclop // ZIP 条目逐项校验路径、类型、去重和解压配额。
func extractZip(root string, data []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return errors.New("invalid ZIP archive")
	}
	if len(reader.File) == 0 || len(reader.File) > maxInputSnapshotEntries {
		return errors.New("input snapshot archive has too many entries")
	}
	var expanded int64
	seen := make(map[string]struct{}, len(reader.File))
	for _, item := range reader.File {
		rel, err := safeArchivePath(item.Name)
		if err != nil {
			return err
		}
		if _, ok := seen[rel]; ok {
			return errors.New("input snapshot archive contains duplicate entries")
		}
		seen[rel] = struct{}{}
		if item.Mode()&os.ModeSymlink != 0 || item.Mode()&os.ModeType != 0 && !item.FileInfo().IsDir() {
			return errors.New("input snapshot archive contains unsupported file")
		}
		if item.FileInfo().IsDir() {
			if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(rel)), managedSnapshotDirectoryMode); err != nil {
				return err
			}
			continue
		}
		if item.UncompressedSize64 > uint64(maxInputSnapshotExpandedBytes) || item.UncompressedSize64 > uint64(maxInputSnapshotExpandedBytes-expanded) {
			return errors.New("input snapshot archive exceeds extracted size limit")
		}
		expanded += int64(item.UncompressedSize64)
		if err := writeArchiveFile(root, rel, item.Open, item.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

//nolint:gocognit,gocyclo,cyclop // TAR 条目校验必须拒绝链接、特殊文件和配额溢出。
func extractTarReader(root string, payload io.Reader) error {
	reader := tar.NewReader(payload)
	var expanded int64
	entries := 0
	seen := map[string]struct{}{}
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return errors.New("invalid TAR archive")
		}
		entries++
		if entries > maxInputSnapshotEntries {
			return errors.New("input snapshot archive has too many entries")
		}
		rel, err := safeArchivePath(header.Name)
		if err != nil {
			return err
		}
		if _, ok := seen[rel]; ok {
			return errors.New("input snapshot archive contains duplicate entries")
		}
		seen[rel] = struct{}{}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(rel)), managedSnapshotDirectoryMode); err != nil {
				return err
			}
		case tar.TypeReg:
			if header.Size < 0 || header.Size > maxInputSnapshotExpandedBytes-expanded {
				return errors.New("input snapshot archive exceeds extracted size limit")
			}
			expanded += header.Size
			if err := writeTarFile(root, rel, reader, header.Mode); err != nil {
				return err
			}
		default:
			return errors.New("input snapshot archive contains unsupported file")
		}
	}
	return nil
}

func safeArchivePath(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	if name == "" || strings.HasPrefix(name, "/") || filepath.IsAbs(name) {
		return "", errors.New("input snapshot archive contains an absolute path")
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("input snapshot archive contains a path traversal")
	}
	return clean, nil
}

func writeArchiveFile(root, rel string, opener func() (io.ReadCloser, error), mode os.FileMode) error {
	reader, err := opener()
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	return writeTarFile(root, rel, reader, int64(mode.Perm()))
}

func writeTarFile(root, rel string, reader io.Reader, mode int64) error {
	destination := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(destination), managedSnapshotDirectoryMode); err != nil {
		return err
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(mode)&archiveFileMode) // #nosec G304,G115 -- rel is normalized beneath the Build-owned extraction root.
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, reader)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func newSnapshotIdentity() string {
	var value [10]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("snapshot_%d", time.Now().UnixNano())
	}
	return "snapshot_" + hex.EncodeToString(value[:])
}
