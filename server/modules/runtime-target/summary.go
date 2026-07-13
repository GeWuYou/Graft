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
	runtimeTargetSummaryTTL     = time.Second
	runtimeTargetSummaryTimeout = 2 * time.Second
	percentageScale             = 100
)

type targetCountMetric struct {
	Available         bool
	Total, Active     int64
	UnavailableReason string
}

type targetImageMetric struct {
	Available         bool
	Total             int64
	UnavailableReason string
}

type targetUsageMetric struct {
	Available             bool
	UsedBytes, TotalBytes int64
	UsagePercent          float64
	UnavailableReason     string
}

type targetRuntimeSummary struct {
	Version, APIVersion string
	Healthy             bool
	CheckedAt           time.Time
	Diagnostic          string
	Workloads           targetCountMetric
	Images              targetImageMetric
	Volumes             targetImageMetric
	Networks            targetImageMetric
	CPU, Memory, Disk   targetUsageMetric
}

type runtimeTargetSnapshotCollector interface {
	Collect(context.Context, store.Target) targetRuntimeSummary
}

type dockerTargetSnapshotCollector struct{}

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

// newSummaryCache 创建一个用于存储摘要并协调进行中采集的空缓存。
func newSummaryCache() *summaryCache {
	return &summaryCache{entries: map[uint64]summaryCacheEntry{}, running: map[uint64]chan struct{}{}}
}

func (c *summaryCache) invalidate(id uint64) {
	c.mu.Lock()
	delete(c.entries, id)
	c.mu.Unlock()
}

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
				return unavailableTargetSummary(ctx.Err().Error(), time.Now().UTC())
			}
		}
		done := make(chan struct{})
		c.running[target.ID] = done
		c.mu.Unlock()

		summary := collectTargetSummary(ctx, target)
		c.mu.Lock()
		delete(c.running, target.ID)
		close(done)
		if summary.Healthy || summary.Workloads.Available || summary.Images.Available || summary.Volumes.Available || summary.Networks.Available || summary.CPU.Available || summary.Memory.Available || summary.Disk.Available {
			c.entries[target.ID] = summaryCacheEntry{summary: summary, expiresAt: time.Now().Add(runtimeTargetSummaryTTL)}
		}
		c.mu.Unlock()
		return summary
	}
}

// unavailableTargetSummary 创建一个运行时摘要，并将所有指标标记为不可用。
// 摘要包含指定的诊断原因和检查时间。
func unavailableTargetSummary(reason string, checkedAt time.Time) targetRuntimeSummary {
	return targetRuntimeSummary{
		CheckedAt:  checkedAt,
		Diagnostic: reason,
		Workloads:  targetCountMetric{UnavailableReason: reason},
		Images:     targetImageMetric{UnavailableReason: reason},
		Volumes:    targetImageMetric{UnavailableReason: reason},
		Networks:   targetImageMetric{UnavailableReason: reason},
		CPU:        targetUsageMetric{UnavailableReason: reason},
		Memory:     targetUsageMetric{UnavailableReason: reason},
		Disk:       targetUsageMetric{UnavailableReason: reason},
	}
}

// runtimeTargetCollector selects the snapshot collector for a runtime provider.
// It returns a Docker collector for the "docker" provider and nil for unsupported providers.
func runtimeTargetCollector(provider string) runtimeTargetSnapshotCollector {
	if provider == "docker" {
		return dockerTargetSnapshotCollector{}
	}
	return nil
}

// collectTargetSummary collects a runtime summary using the target's configured provider.
// It returns an unavailable summary when no collector is registered for the provider.
func collectTargetSummary(ctx context.Context, target store.Target) targetRuntimeSummary {
	collector := runtimeTargetCollector(target.Provider)
	if collector == nil {
		return unavailableTargetSummary("Runtime provider is unavailable", time.Now().UTC())
	}
	return collector.Collect(ctx, target)
}

func (dockerTargetSnapshotCollector) Collect(ctx context.Context, target store.Target) targetRuntimeSummary {
	checkedAt := time.Now().UTC()
	if !target.Availability || target.ConnectionKind != "unix_socket" {
		diagnostic := target.LastError
		if diagnostic == "" {
			diagnostic = "Docker target is unavailable"
		}
		return unavailableTargetSummary(diagnostic, checkedAt)
	}
	client, err := mobyclient.New(mobyclient.WithHost(target.EndpointLabel))
	if err != nil {
		return unavailableTargetSummary("Docker client is unavailable", checkedAt)
	}
	defer closeDockerClient(client)
	collectCtx, cancel := context.WithTimeout(ctx, runtimeTargetSummaryTimeout)
	defer cancel()
	version, err := client.ServerVersion(collectCtx, mobyclient.ServerVersionOptions{})
	if err != nil {
		return unavailableTargetSummary("Docker target is unavailable", checkedAt)
	}
	result := targetRuntimeSummary{Healthy: true, CheckedAt: checkedAt, Version: version.Version, APIVersion: version.APIVersion}
	result.Workloads = collectContainerMetric(collectCtx, client)
	result.Images = collectImageMetric(collectCtx, client)
	result.Volumes = collectVolumeMetric(collectCtx, client)
	result.Networks = collectNetworkMetric(collectCtx, client)
	result.CPU, result.Memory = collectHostUsage(collectCtx, collectHostCPUUsage, collectHostMemoryUsage)
	result.Disk = collectDockerFilesystemUsage(collectCtx, client)
	return result
}

// closeDockerClient closes the Docker client.
func closeDockerClient(client *mobyclient.Client) {
	_ = client.Close()
}

// collectContainerMetric collects Docker workload counts and identifies active containers.
// It marks the metric unavailable when Docker cannot provide the container list.
func collectContainerMetric(ctx context.Context, client *mobyclient.Client) targetCountMetric {
	containers, err := client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true})
	if err != nil {
		return targetCountMetric{UnavailableReason: "Docker workload metrics are unavailable"}
	}
	metric := targetCountMetric{Available: true, Total: int64(len(containers.Items))}
	for _, item := range containers.Items {
		if string(item.State) == "running" {
			metric.Active++
		}
	}
	return metric
}

// collectImageMetric collects the total number of Docker images.
// The returned metric is marked unavailable when the Docker image query fails.
func collectImageMetric(ctx context.Context, client *mobyclient.Client) targetImageMetric {
	images, err := client.ImageList(ctx, mobyclient.ImageListOptions{All: true})
	if err != nil {
		return targetImageMetric{UnavailableReason: "Docker image metrics are unavailable"}
	}
	return targetImageMetric{Available: true, Total: int64(len(images.Items))}
}

// collectVolumeMetric collects the total number of Docker volumes.
//
// The returned metric is marked unavailable when Docker cannot provide the volume list.
func collectVolumeMetric(ctx context.Context, client *mobyclient.Client) targetImageMetric {
	volumes, err := client.VolumeList(ctx, mobyclient.VolumeListOptions{})
	if err != nil {
		return targetImageMetric{UnavailableReason: "Docker volume metrics are unavailable"}
	}
	return targetImageMetric{Available: true, Total: int64(len(volumes.Items))}
}

// collectNetworkMetric collects the total number of Docker networks.
//
// The returned metric is marked unavailable when the Docker network list cannot be retrieved.
func collectNetworkMetric(ctx context.Context, client *mobyclient.Client) targetImageMetric {
	networks, err := client.NetworkList(ctx, mobyclient.NetworkListOptions{})
	if err != nil {
		return targetImageMetric{UnavailableReason: "Docker network metrics are unavailable"}
	}
	return targetImageMetric{Available: true, Total: int64(len(networks.Items))}
}

// collectDockerFilesystemUsage collects usage metrics for Docker's data directory filesystem.
// It marks the metrics unavailable when Docker information or filesystem statistics cannot be obtained.
func collectDockerFilesystemUsage(ctx context.Context, client *mobyclient.Client) targetUsageMetric {
	if err := ctx.Err(); err != nil {
		return targetUsageMetric{UnavailableReason: "Docker data directory filesystem is unavailable"}
	}
	info, infoErr := client.Info(ctx, mobyclient.InfoOptions{})
	var fs syscall.Statfs_t
	if infoErr == nil && info.Info.DockerRootDir != "" && syscall.Statfs(info.Info.DockerRootDir, &fs) == nil {
		return filesystemUsageMetric(fs.Blocks, fs.Bfree, fs.Bsize)
	}
	return targetUsageMetric{UnavailableReason: "Docker data directory filesystem is unavailable"}
}

type hostUsageCollector func(context.Context) targetUsageMetric

// collectHostUsage collects CPU and memory usage metrics.
func collectHostUsage(ctx context.Context, collectCPU, collectMemory hostUsageCollector) (targetUsageMetric, targetUsageMetric) {
	return collectCPU(ctx), collectMemory(ctx)
}

// collectHostCPUUsage 获取主机 CPU 使用率指标。
// CPU 指标无法获取时返回标记为不可用的指标。
func collectHostCPUUsage(ctx context.Context) targetUsageMetric {
	values, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil || len(values) == 0 {
		return targetUsageMetric{UnavailableReason: "Host CPU metrics are unavailable"}
	}
	return targetUsageMetric{Available: true, UsagePercent: values[0]}
}

// collectHostMemoryUsage collects host memory usage metrics.
//
// It returns the used and total memory, along with the usage percentage. The
// metrics are marked unavailable when the memory snapshot cannot be collected
// or has no total memory.
func collectHostMemoryUsage(ctx context.Context) targetUsageMetric {
	snapshot, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil || snapshot == nil || snapshot.Total == 0 {
		return targetUsageMetric{UnavailableReason: "Host memory metrics are unavailable"}
	}
	return targetUsageMetric{Available: true, UsedBytes: uint64ToInt64(snapshot.Used), TotalBytes: uint64ToInt64(snapshot.Total), UsagePercent: snapshot.UsedPercent}
}

// uint64ToInt64 converts a uint64 value to int64, clamping values above the maximum int64 value.
func uint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}

// filesystemUsageMetric 根据文件系统块数计算已用容量、总容量和使用率；输入无效时返回不可用指标。
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
	return targetUsageMetric{Available: true, UsedBytes: used, TotalBytes: total, UsagePercent: float64(used) * percentageScale / float64(total)}
}

// filesystemBytes converts a block count and block size to bytes when the inputs produce a valid int64 value.
// It returns false when the block size is non-positive or the result would overflow int64.
func filesystemBytes(blocks uint64, blockSize int64) (int64, bool) {
	if blockSize <= 0 || blocks > uint64(math.MaxInt64)/uint64(blockSize) {
		return 0, false
	}
	//nolint:gosec // The overflow guard above proves this product fits in int64.
	return int64(blocks * uint64(blockSize)), true
}
