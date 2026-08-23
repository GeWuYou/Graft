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

var errInvalidInputSnapshotUpload = errors.New("invalid input snapshot upload")

func invalidInputSnapshotUpload(message string) error {
	return fmt.Errorf("%w: %s", errInvalidInputSnapshotUpload, message)
}

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
		return moduleapi.WorkspaceSnapshot{}, invalidInputSnapshotUpload("input snapshot must contain a root Dockerfile")
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
		_ = releaseMaterialization(ctx, snapshot.MaterializationRef)
	}
	return created, nil
}

func readBoundedArchive(reader io.Reader, declaredSize int64) ([]byte, error) {
	if declaredSize > maxInputSnapshotUploadBytes {
		return nil, invalidInputSnapshotUpload("input snapshot archive exceeds upload limit")
	}
	limited := io.LimitReader(reader, maxInputSnapshotUploadBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read input snapshot archive: %w", err)
	}
	if int64(len(data)) > maxInputSnapshotUploadBytes {
		return nil, invalidInputSnapshotUpload("input snapshot archive exceeds upload limit")
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
	var expandedStream *countingReader
	if len(data) >= 2 && data[0] == 'P' && data[1] == 'K' {
		err = extractZip(root, data)
	} else {
		payload := io.Reader(bytes.NewReader(data))
		if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
			gzipReader, gzipErr := gzip.NewReader(bytes.NewReader(data))
			if gzipErr != nil {
				return "", invalidInputSnapshotUpload("invalid TAR.GZ archive")
			}
			// 直接将解压流交给 TAR reader；写入端按实际字节执行展开配额，避免再复制一份完整归档。
			defer func() { _ = gzipReader.Close() }()
			expandedStream = &countingReader{Reader: io.LimitReader(gzipReader, maxInputSnapshotExpandedBytes+1)}
			payload = expandedStream
		}
		err = extractTarReader(root, payload)
	}
	if expandedStream != nil && expandedStream.n > maxInputSnapshotExpandedBytes {
		return "", invalidInputSnapshotUpload("input snapshot archive exceeds extracted size limit")
	}
	if err != nil {
		return "", err
	}
	failed = false
	return root, nil
}

type countingReader struct {
	io.Reader
	n int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.n += int64(n)
	return n, err
}

//nolint:gocognit,gocyclo,cyclop // ZIP 条目逐项校验路径、类型、去重和解压配额。
func extractZip(root string, data []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return invalidInputSnapshotUpload("invalid ZIP archive")
	}
	if len(reader.File) == 0 || len(reader.File) > maxInputSnapshotEntries {
		return invalidInputSnapshotUpload("input snapshot archive has too many entries")
	}
	var expanded int64
	seen := make(map[string]struct{}, len(reader.File))
	for _, item := range reader.File {
		rel, err := safeArchivePath(item.Name)
		if err != nil {
			return err
		}
		if _, ok := seen[rel]; ok {
			return invalidInputSnapshotUpload("input snapshot archive contains duplicate entries")
		}
		seen[rel] = struct{}{}
		if item.Mode()&os.ModeSymlink != 0 || item.Mode()&os.ModeType != 0 && !item.FileInfo().IsDir() {
			return invalidInputSnapshotUpload("input snapshot archive contains unsupported file")
		}
		if item.FileInfo().IsDir() {
			destination, destinationErr := archiveDestination(root, rel)
			if destinationErr != nil {
				return destinationErr
			}
			if err := os.MkdirAll(destination, managedSnapshotDirectoryMode); err != nil {
				return err
			}
			continue
		}
		if item.UncompressedSize64 > uint64(maxInputSnapshotExpandedBytes) || item.UncompressedSize64 > uint64(maxInputSnapshotExpandedBytes-expanded) {
			return invalidInputSnapshotUpload("input snapshot archive exceeds extracted size limit")
		}
		remaining := maxInputSnapshotExpandedBytes - expanded
		if remaining <= 0 || item.UncompressedSize64 > uint64(remaining) {
			return invalidInputSnapshotUpload("input snapshot archive exceeds extracted size limit")
		}
		written, err := writeArchiveFile(root, rel, item.Open, item.Mode().Perm(), remaining)
		if err != nil {
			return err
		}
		expanded += written
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
			return invalidInputSnapshotUpload("invalid TAR archive")
		}
		entries++
		if entries > maxInputSnapshotEntries {
			return invalidInputSnapshotUpload("input snapshot archive has too many entries")
		}
		rel, err := safeArchivePath(header.Name)
		if err != nil {
			return err
		}
		if _, ok := seen[rel]; ok {
			return invalidInputSnapshotUpload("input snapshot archive contains duplicate entries")
		}
		seen[rel] = struct{}{}
		switch header.Typeflag {
		case tar.TypeDir:
			destination, destinationErr := archiveDestination(root, rel)
			if destinationErr != nil {
				return destinationErr
			}
			if err := os.MkdirAll(destination, managedSnapshotDirectoryMode); err != nil {
				return err
			}
		case tar.TypeReg:
			if header.Size < 0 || header.Size > maxInputSnapshotExpandedBytes-expanded {
				return invalidInputSnapshotUpload("input snapshot archive exceeds extracted size limit")
			}
			written, err := writeTarFile(root, rel, reader, header.Mode, maxInputSnapshotExpandedBytes-expanded)
			if err != nil {
				return err
			}
			expanded += written
		default:
			return invalidInputSnapshotUpload("input snapshot archive contains unsupported file")
		}
	}
	return nil
}

func safeArchivePath(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	if invalidArchivePath(name) {
		return "", invalidInputSnapshotUpload("input snapshot archive contains an absolute path")
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", invalidInputSnapshotUpload("input snapshot archive contains a path traversal")
	}
	return clean, nil
}

func invalidArchivePath(name string) bool {
	return name == "" || strings.ContainsRune(name, 0) || strings.HasPrefix(name, "/") || filepath.IsAbs(name) || filepath.VolumeName(name) != "" || hasDrivePrefix(name)
}

func hasDrivePrefix(name string) bool {
	return len(name) >= 2 && name[1] == ':'
}

func writeArchiveFile(root, rel string, opener func() (io.ReadCloser, error), mode os.FileMode, limit int64) (int64, error) {
	reader, err := opener()
	if err != nil {
		return 0, err
	}
	defer func() { _ = reader.Close() }()
	return writeTarFile(root, rel, reader, int64(mode.Perm()), limit)
}

func writeTarFile(root, rel string, reader io.Reader, mode int64, limit int64) (int64, error) {
	if limit < 0 {
		return 0, invalidInputSnapshotUpload("input snapshot archive exceeds extracted size limit")
	}
	destination, err := archiveDestination(root, rel)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(destination), managedSnapshotDirectoryMode); err != nil {
		return 0, err
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(mode)&archiveFileMode) // #nosec G304,G115 -- destination is containment-checked and archive mode is masked to os.FileMode.
	if err != nil {
		return 0, err
	}
	written, copyErr := io.Copy(file, io.LimitReader(reader, limit+1))
	closeErr := file.Close()
	if copyErr != nil {
		return written, copyErr
	}
	if written > limit {
		return written, invalidInputSnapshotUpload("input snapshot archive exceeds extracted size limit")
	}
	return written, closeErr
}

func archiveDestination(root, rel string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	destination, err := filepath.Abs(filepath.Join(rootAbs, filepath.FromSlash(rel)))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootAbs, destination)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", invalidInputSnapshotUpload("input snapshot archive contains a path traversal")
	}
	return destination, nil
}

func newSnapshotIdentity() string {
	var value [10]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("snapshot_%d", time.Now().UnixNano())
	}
	return "snapshot_" + hex.EncodeToString(value[:])
}
