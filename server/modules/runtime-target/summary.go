package runtimetarget

import (
	"context"
	"math"
	"sync"
	"syscall"
	"time"

	mobyclient "github.com/moby/moby/client"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	store "graft/server/modules/runtime-target/store"
)

const (
	runtimeTargetSummaryTTL = time.Second
	percentageScale         = 100
)

type targetCountMetric struct {
	Available               bool
	Total, Running, Stopped int64
	UnavailableReason       string
}
type targetImageMetric struct {
	Available           bool
	Total, Used, Unused int64
	UnavailableReason   string
}
type targetUsageMetric struct {
	Available             bool
	UsedBytes, TotalBytes int64
	UsagePercent          float64
	UnavailableReason     string
}
type targetRuntimeSummary struct {
	Containers        targetCountMetric
	Images            targetImageMetric
	CPU, Memory, Disk targetUsageMetric
}
type summaryCacheEntry struct {
	summary   targetRuntimeSummary
	expiresAt time.Time
}

// summaryCache keeps an intentionally short, process-local snapshot and collapses concurrent collection.
type summaryCache struct {
	mu      sync.Mutex
	entries map[uint64]summaryCacheEntry
	running map[uint64]chan struct{}
}

func newSummaryCache() *summaryCache {
	return &summaryCache{entries: map[uint64]summaryCacheEntry{}, running: map[uint64]chan struct{}{}}
}
func (c *summaryCache) invalidate(id uint64) { c.mu.Lock(); delete(c.entries, id); c.mu.Unlock() }

//nolint:cyclop // Cache wait, cache hit, and one collector path are intentionally colocated for correctness.
func (c *summaryCache) get(ctx context.Context, target store.Target) targetRuntimeSummary {
	for {
		c.mu.Lock()
		if entry, ok := c.entries[target.ID]; ok && time.Now().Before(entry.expiresAt) {
			c.mu.Unlock()
			return entry.summary
		}
		if done, ok := c.running[target.ID]; ok {
			c.mu.Unlock()
			select {
			case <-done:
				continue
			case <-ctx.Done():
				return unavailableTargetSummary(ctx.Err().Error())
			}
		}
		done := make(chan struct{})
		c.running[target.ID] = done
		c.mu.Unlock()
		summary := collectTargetSummary(ctx, target)
		c.mu.Lock()
		delete(c.running, target.ID)
		close(done)
		// Errors are intentionally not cached: a later request must not receive stale metrics.
		if summary.Containers.Available || summary.Images.Available || summary.CPU.Available || summary.Memory.Available || summary.Disk.Available {
			c.entries[target.ID] = summaryCacheEntry{summary: summary, expiresAt: time.Now().Add(runtimeTargetSummaryTTL)}
		}
		c.mu.Unlock()
		return summary
	}
}

func unavailableTargetSummary(reason string) targetRuntimeSummary {
	return targetRuntimeSummary{Containers: targetCountMetric{UnavailableReason: reason}, Images: targetImageMetric{UnavailableReason: reason}, CPU: targetUsageMetric{UnavailableReason: reason}, Memory: targetUsageMetric{UnavailableReason: reason}, Disk: targetUsageMetric{UnavailableReason: reason}}
}

//nolint:funlen,gocognit,gocyclo,cyclop // Docker summary collection deliberately preserves each metric's independent availability.
func collectTargetSummary(ctx context.Context, target store.Target) targetRuntimeSummary {
	if !target.Availability || target.Provider != "docker" || target.ConnectionKind != "unix_socket" {
		return unavailableTargetSummary("Docker target is unavailable")
	}
	client, err := mobyclient.New(mobyclient.WithHost(target.EndpointLabel))
	if err != nil {
		return unavailableTargetSummary("Docker client is unavailable")
	}
	containers, err := client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true})
	if err != nil {
		return unavailableTargetSummary("Docker container metrics are unavailable")
	}
	result := unavailableTargetSummary("Docker metric is unavailable")
	result.Containers = targetCountMetric{Available: true, Total: int64(len(containers.Items))}
	for _, item := range containers.Items {
		if string(item.State) == "running" {
			result.Containers.Running++
		} else {
			result.Containers.Stopped++
		}
	}
	images, imageErr := client.ImageList(ctx, mobyclient.ImageListOptions{All: true})
	if imageErr == nil {
		result.Images = targetImageMetric{Available: true, Total: int64(len(images.Items))}
		for _, image := range images.Items {
			if image.Containers > 0 {
				result.Images.Used++
			}
		}
		result.Images.Unused = result.Images.Total - result.Images.Used
	} else {
		result.Images.UnavailableReason = "Docker image metrics are unavailable"
	}
	result.CPU = collectHostCPUUsage(ctx)
	result.Memory = collectHostMemoryUsage(ctx)
	info, infoErr := client.Info(ctx, mobyclient.InfoOptions{})
	var fs syscall.Statfs_t
	if infoErr == nil && info.Info.DockerRootDir != "" && syscall.Statfs(info.Info.DockerRootDir, &fs) == nil {
		total := filesystemBytes(fs.Blocks, fs.Bsize)
		used := total - filesystemBytes(fs.Bfree, fs.Bsize)
		result.Disk = targetUsageMetric{Available: true, UsedBytes: used, TotalBytes: total}
		if total > 0 {
			result.Disk.UsagePercent = float64(used) * percentageScale / float64(total)
		}
	} else {
		result.Disk.UnavailableReason = "Docker data directory filesystem is unavailable"
	}
	return result
}

func collectHostCPUUsage(ctx context.Context) targetUsageMetric {
	values, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil || len(values) == 0 {
		return targetUsageMetric{UnavailableReason: "Host CPU metrics are unavailable"}
	}
	return targetUsageMetric{Available: true, UsagePercent: values[0]}
}

func collectHostMemoryUsage(ctx context.Context) targetUsageMetric {
	snapshot, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil || snapshot == nil || snapshot.Total == 0 {
		return targetUsageMetric{UnavailableReason: "Host memory metrics are unavailable"}
	}
	return targetUsageMetric{Available: true, UsedBytes: uint64ToInt64(snapshot.Used), TotalBytes: uint64ToInt64(snapshot.Total), UsagePercent: snapshot.UsedPercent}
}

func uint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}

func filesystemBytes(blocks uint64, blockSize int64) int64 {
	if blocks == 0 || blockSize <= 0 || blocks > uint64(math.MaxInt64)/uint64(blockSize) {
		return math.MaxInt64
	}
	//nolint:gosec // The overflow guard above proves this value fits in int64.
	return int64(blocks * uint64(blockSize))
}
