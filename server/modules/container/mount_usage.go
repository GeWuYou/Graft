// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
	var total int64
	err := filepath.WalkDir(root, func(_ string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, mapMountUsageScanError(err)
	}
	return total, nil
}

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

func mountUsageCacheKey(ref Ref, mountID string) string {
	return strings.TrimSpace(ref.Value) + "\x00" + strings.TrimSpace(mountID)
}

func formatIECBytes(size int64) string {
	if size < 0 {
		size = 0
	}
	const unit = int64(1024)
	switch {
	case size < unit*unit:
		return formatIECValue(float64(size)/float64(unit), "KiB")
	case size < unit*unit*unit:
		return formatIECValue(float64(size)/float64(unit*unit), "MiB")
	default:
		return formatIECValue(float64(size)/float64(unit*unit*unit), "GiB")
	}
}

func formatIECValue(value float64, suffix string) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%.0f %s", value, suffix)
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}
