// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
	"time"
)

func TestFilesystemMountUsageScannerSkipsDeletedChildren(t *testing.T) {
	t.Parallel()

	size, err := scanMountUsageFS(context.Background(), disappearingChildFS{}, ".")
	if err != nil {
		t.Fatalf("scan existing root with child deletion: %v", err)
	}
	if size != 4 {
		t.Fatalf("expected scan to keep existing file size, got %d", size)
	}
}

func TestFilesystemMountUsageScannerMapsOnlyMissingRootToMountNotFound(t *testing.T) {
	t.Parallel()

	scanner := filesystemMountUsageScanner{}
	_, err := scanner.ScanUsage(context.Background(), filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, errContainerMountNotFound) {
		t.Fatalf("expected missing root to map to mount not found, got %v", err)
	}
}

type disappearingChildFS struct{}

func (disappearingChildFS) Open(name string) (fs.File, error) {
	switch name {
	case ".":
		return &staticDir{
			info: staticFileInfo{name: ".", dir: true},
			entries: []fs.DirEntry{
				staticDirEntry{info: staticFileInfo{name: "kept.txt", size: 4}},
				staticDirEntry{info: staticFileInfo{name: "gone", dir: true}},
			},
		}, nil
	case "kept.txt":
		return staticFile{info: staticFileInfo{name: "kept.txt", size: 4}}, nil
	case "gone":
		return nil, fs.ErrNotExist
	default:
		return nil, fs.ErrNotExist
	}
}

type staticDir struct {
	info    staticFileInfo
	entries []fs.DirEntry
	read    bool
}

func (d *staticDir) Stat() (fs.FileInfo, error) { return d.info, nil }
func (d *staticDir) Close() error               { return nil }
func (d *staticDir) Read([]byte) (int, error)   { return 0, fs.ErrInvalid }

func (d *staticDir) ReadDir(int) ([]fs.DirEntry, error) {
	if d.read {
		return nil, nil
	}
	d.read = true
	return d.entries, nil
}

type staticFile struct {
	info staticFileInfo
}

func (f staticFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f staticFile) Close() error               { return nil }
func (f staticFile) Read([]byte) (int, error)   { return 0, fs.ErrClosed }

type staticDirEntry struct {
	info staticFileInfo
}

func (e staticDirEntry) Name() string               { return e.info.name }
func (e staticDirEntry) IsDir() bool                { return e.info.dir }
func (e staticDirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e staticDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

type staticFileInfo struct {
	name string
	size int64
	dir  bool
}

func (i staticFileInfo) Name() string { return i.name }
func (i staticFileInfo) Size() int64  { return i.size }
func (i staticFileInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o755
	}
	return 0o644
}
func (i staticFileInfo) ModTime() time.Time { return time.Time{} }
func (i staticFileInfo) IsDir() bool        { return i.dir }
func (i staticFileInfo) Sys() any           { return nil }
