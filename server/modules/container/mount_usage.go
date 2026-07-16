package container

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	containerMountUsageTimeout  = 4 * time.Second
	containerMountUsageCacheTTL = 45 * time.Second
)

type mountUsageScanner interface {
	ScanUsage(ctx context.Context, path string) (int64, error)
}

type filesystemMountUsageScanner struct{}

func (filesystemMountUsageScanner) ScanUsage(ctx context.Context, root string) (int64, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return 0, errMountUsageUnsupported
	}
	info, err := os.Stat(root)
	if err != nil {
		return 0, mapMountUsageScanError(err)
	}
	if !info.IsDir() {
		if info.Mode().IsRegular() {
			return info.Size(), nil
		}
		return 0, errMountUsageUnsupported
	}
	total, err := scanMountUsageFS(ctx, os.DirFS(root), ".")
	if err != nil {
		return 0, mapMountUsageScanError(err)
	}
	return total, nil
}

func scanMountUsageFS(ctx context.Context, fileSystem fs.FS, root string) (int64, error) {
	var total int64
	err := fs.WalkDir(fileSystem, root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return handleMountUsageWalkError(root, path, entry, walkErr)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return handleMountUsageEntryInfoError(err)
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}

func handleMountUsageWalkError(root string, path string, entry fs.DirEntry, err error) error {
	if path == root {
		return err
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if entry != nil && entry.IsDir() {
		return filepath.SkipDir
	}
	return nil
}

func handleMountUsageEntryInfoError(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// mapMountUsageScanError 将文件系统和上下文错误映射为容器运行时错误。
func mapMountUsageScanError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return errContainerRuntimeTimeout
	case errors.Is(err, os.ErrNotExist):
		return errContainerMountNotFound
	case errors.Is(err, os.ErrPermission):
		return errRuntimePermissionDenied
	default:
		return err
	}
}

type mountUsageCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	now   func() time.Time
	items map[string]mountUsageCacheEntry
}

type mountUsageCacheEntry struct {
	usage     MountUsage
	expiresAt time.Time
}

// newMountUsageCache 创建挂载使用量缓存；TTL 为零或负数时使用包级默认值。
func newMountUsageCache(ttl time.Duration) *mountUsageCache {
	if ttl <= 0 {
		ttl = containerMountUsageCacheTTL
	}
	return &mountUsageCache{
		ttl:   ttl,
		now:   time.Now,
		items: make(map[string]mountUsageCacheEntry),
	}
}

func (c *mountUsageCache) get(key string) (MountUsage, bool) {
	if c == nil {
		return MountUsage{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[key]
	if !ok {
		return MountUsage{}, false
	}
	if !c.now().Before(entry.expiresAt) {
		delete(c.items, key)
		return MountUsage{}, false
	}
	usage := entry.usage
	usage.Cached = true
	return usage, true
}

func (c *mountUsageCache) set(key string, usage MountUsage) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	usage.Cached = false
	c.items[key] = mountUsageCacheEntry{
		usage:     usage,
		expiresAt: c.now().Add(c.ttl),
	}
}

// mountUsageCacheKey 使用容器引用和挂载 ID 生成挂载使用量查询键，避免不同容器的同名挂载共享缓存。
func mountUsageCacheKey(ref Ref, mountID string) string {
	return strings.TrimSpace(ref.Value) + "\x00" + strings.TrimSpace(mountID)
}

// formatIECBytes 将字节数格式化为易读的 IEC 二进制单位；负数按零处理。
func formatIECBytes(size int64) string {
	if size < 0 {
		size = 0
	}
	const unit = int64(1024)
	switch {
	case size < unit:
		return fmt.Sprintf("%d B", size)
	case size < unit*unit:
		return formatIECValue(float64(size)/float64(unit), "KiB")
	case size < unit*unit*unit:
		return formatIECValue(float64(size)/float64(unit*unit), "MiB")
	default:
		return formatIECValue(float64(size)/float64(unit*unit*unit), "GiB")
	}
}

// formatIECValue 按整数零位、小数一位的规则格式化数值，并追加指定单位后缀。
func formatIECValue(value float64, suffix string) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%.0f %s", value, suffix)
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}
