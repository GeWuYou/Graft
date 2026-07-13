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

// newSummaryCache creates an empty summary cache with initialized cache and in-flight collection state.
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

// unavailableTargetSummary 创建一个运行时摘要，并为所有指标设置不可用原因。
// reason 指示这些指标不可用的原因。
func unavailableTargetSummary(reason string) targetRuntimeSummary {
	return targetRuntimeSummary{Containers: targetCountMetric{UnavailableReason: reason}, Images: targetImageMetric{UnavailableReason: reason}, CPU: targetUsageMetric{UnavailableReason: reason}, Memory: targetUsageMetric{UnavailableReason: reason}, Disk: targetUsageMetric{UnavailableReason: reason}}
}

// collectTargetSummary collects container, image, CPU, memory, and disk metrics for a Docker target, preserving availability for each metric independently.
func collectTargetSummary(ctx context.Context, target store.Target) targetRuntimeSummary {
	if !target.Availability || target.Provider != "docker" || target.ConnectionKind != "unix_socket" {
		return unavailableTargetSummary("Docker target is unavailable")
	}
	client, err := mobyclient.New(mobyclient.WithHost(target.EndpointLabel))
	if err != nil {
		return unavailableTargetSummary("Docker client is unavailable")
	}
	defer closeDockerClient(client)
	containers, err := collectContainerMetric(ctx, client)
	if err != nil {
		return unavailableTargetSummary("Docker container metrics are unavailable")
	}
	result := unavailableTargetSummary("Docker metric is unavailable")
	result.Containers = containers
	result.Images = collectImageMetric(ctx, client)
	result.CPU, result.Memory = collectHostUsage(ctx, collectHostCPUUsage, collectHostMemoryUsage)
	result.Disk = collectDockerFilesystemUsage(ctx, client)
	return result
}

// closeDockerClient releases idle client connections after a collection attempt.
func closeDockerClient(client *mobyclient.Client) {
	_ = client.Close()
}

func collectContainerMetric(ctx context.Context, client *mobyclient.Client) (targetCountMetric, error) {
	containers, err := client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true})
	if err != nil {
		return targetCountMetric{}, err
	}
	metric := targetCountMetric{Available: true, Total: int64(len(containers.Items))}
	for _, item := range containers.Items {
		if string(item.State) == "running" {
			metric.Running++
		} else {
			metric.Stopped++
		}
	}
	return metric, nil
}

func collectImageMetric(ctx context.Context, client *mobyclient.Client) targetImageMetric {
	images, err := client.ImageList(ctx, mobyclient.ImageListOptions{All: true})
	if err != nil {
		return targetImageMetric{UnavailableReason: "Docker image metrics are unavailable"}
	}
	metric := targetImageMetric{Available: true, Total: int64(len(images.Items))}
	for _, image := range images.Items {
		if image.Containers > 0 {
			metric.Used++
		}
	}
	metric.Unused = metric.Total - metric.Used
	return metric
}

func collectDockerFilesystemUsage(ctx context.Context, client *mobyclient.Client) targetUsageMetric {
	info, infoErr := client.Info(ctx, mobyclient.InfoOptions{})
	var fs syscall.Statfs_t
	if infoErr == nil && info.Info.DockerRootDir != "" && syscall.Statfs(info.Info.DockerRootDir, &fs) == nil {
		return filesystemUsageMetric(fs.Blocks, fs.Bfree, fs.Bsize)
	}
	return targetUsageMetric{UnavailableReason: "Docker data directory filesystem is unavailable"}
}

type hostUsageCollector func(context.Context) targetUsageMetric

// collectHostUsage runs the independent host collectors without letting one unavailable metric affect the other.
func collectHostUsage(ctx context.Context, collectCPU, collectMemory hostUsageCollector) (targetUsageMetric, targetUsageMetric) {
	return collectCPU(ctx), collectMemory(ctx)
}

// collectHostCPUUsage collects the host CPU usage percentage.
// It returns an unavailable metric when the CPU usage cannot be retrieved.
func collectHostCPUUsage(ctx context.Context) targetUsageMetric {
	values, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil || len(values) == 0 {
		return targetUsageMetric{UnavailableReason: "Host CPU metrics are unavailable"}
	}
	return targetUsageMetric{Available: true, UsagePercent: values[0]}
}

// collectHostMemoryUsage collects host memory usage metrics.
// It returns an unavailable metric when the memory snapshot cannot be obtained or has no total capacity.
func collectHostMemoryUsage(ctx context.Context) targetUsageMetric {
	snapshot, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil || snapshot == nil || snapshot.Total == 0 {
		return targetUsageMetric{UnavailableReason: "Host memory metrics are unavailable"}
	}
	return targetUsageMetric{Available: true, UsedBytes: uint64ToInt64(snapshot.Used), TotalBytes: uint64ToInt64(snapshot.Total), UsagePercent: snapshot.UsedPercent}
}

// uint64ToInt64 converts a uint64 value to int64, saturating values greater than
// math.MaxInt64 at math.MaxInt64.
func uint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}

// filesystemUsageMetric calculates filesystem usage while preserving zero free blocks as a valid full filesystem.
func filesystemUsageMetric(blocks, freeBlocks uint64, blockSize int64) targetUsageMetric {
	total, valid := filesystemBytes(blocks, blockSize)
	if !valid || total == 0 {
		return targetUsageMetric{UnavailableReason: "Docker data directory filesystem is unavailable"}
	}
	free, valid := filesystemBytes(freeBlocks, blockSize)
	if !valid || freeBlocks > blocks {
		return targetUsageMetric{UnavailableReason: "Docker data directory filesystem is unavailable"}
	}
	used := total - free
	return targetUsageMetric{
		Available:    true,
		UsedBytes:    used,
		TotalBytes:   total,
		UsagePercent: float64(used) * percentageScale / float64(total),
	}
}

// filesystemBytes converts filesystem block counts to bytes and reports whether the value fits in int64.
func filesystemBytes(blocks uint64, blockSize int64) (int64, bool) {
	if blockSize <= 0 || blocks > uint64(math.MaxInt64)/uint64(blockSize) {
		return 0, false
	}
	//nolint:gosec // The overflow guard above proves this value fits in int64.
	return int64(blocks * uint64(blockSize)), true
}
