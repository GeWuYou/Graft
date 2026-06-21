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

// MapMountUsageScanError translates filesystem and context errors to container runtime errors.
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

// newMountUsageCache creates a new mount usage cache with the specified TTL, or the default TTL if the provided value is zero or negative.
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

// mountUsageCacheKey produces a cache key for mount usage lookup using the given reference and mount ID.
func mountUsageCacheKey(ref Ref, mountID string) string {
	return strings.TrimSpace(ref.Value) + "\x00" + strings.TrimSpace(mountID)
}

// formatIECBytes formats a byte count as a human-readable IEC binary string using KiB, MiB, or GiB units as appropriate. Negative sizes are treated as zero.
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

// formatIECValue formats a numeric value to a string, using zero decimal places for integers and one decimal place for non-integers, followed by the provided suffix.
func formatIECValue(value float64, suffix string) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%.0f %s", value, suffix)
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}
