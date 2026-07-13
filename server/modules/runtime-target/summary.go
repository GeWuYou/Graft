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

// newSummaryCache creates an empty summary cache with initialized cache and in-flight collection state.
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

func runtimeTargetCollector(provider string) runtimeTargetSnapshotCollector {
	if provider == "docker" {
		return dockerTargetSnapshotCollector{}
	}
	return nil
}

// collectTargetSummary delegates collection to the provider-local snapshot seam.
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
	version, err := client.ServerVersion(ctx, mobyclient.ServerVersionOptions{})
	if err != nil {
		return unavailableTargetSummary("Docker target is unavailable", checkedAt)
	}
	result := targetRuntimeSummary{Healthy: true, CheckedAt: checkedAt, Version: version.Version, APIVersion: version.APIVersion}
	result.Workloads = collectContainerMetric(ctx, client)
	result.Images = collectImageMetric(ctx, client)
	result.Volumes = collectVolumeMetric(ctx, client)
	result.Networks = collectNetworkMetric(ctx, client)
	result.CPU, result.Memory = collectHostUsage(ctx, collectHostCPUUsage, collectHostMemoryUsage)
	result.Disk = collectDockerFilesystemUsage(ctx, client)
	return result
}

func closeDockerClient(client *mobyclient.Client) {
	_ = client.Close()
}

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

func collectImageMetric(ctx context.Context, client *mobyclient.Client) targetImageMetric {
	images, err := client.ImageList(ctx, mobyclient.ImageListOptions{All: true})
	if err != nil {
		return targetImageMetric{UnavailableReason: "Docker image metrics are unavailable"}
	}
	return targetImageMetric{Available: true, Total: int64(len(images.Items))}
}

func collectVolumeMetric(ctx context.Context, client *mobyclient.Client) targetImageMetric {
	volumes, err := client.VolumeList(ctx, mobyclient.VolumeListOptions{})
	if err != nil {
		return targetImageMetric{UnavailableReason: "Docker volume metrics are unavailable"}
	}
	return targetImageMetric{Available: true, Total: int64(len(volumes.Items))}
}

func collectNetworkMetric(ctx context.Context, client *mobyclient.Client) targetImageMetric {
	networks, err := client.NetworkList(ctx, mobyclient.NetworkListOptions{})
	if err != nil {
		return targetImageMetric{UnavailableReason: "Docker network metrics are unavailable"}
	}
	return targetImageMetric{Available: true, Total: int64(len(networks.Items))}
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

func collectHostUsage(ctx context.Context, collectCPU, collectMemory hostUsageCollector) (targetUsageMetric, targetUsageMetric) {
	return collectCPU(ctx), collectMemory(ctx)
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

func filesystemBytes(blocks uint64, blockSize int64) (int64, bool) {
	if blockSize <= 0 || blocks > uint64(math.MaxInt64)/uint64(blockSize) {
		return 0, false
	}
	//nolint:gosec // The overflow guard above proves this product fits in int64.
	return int64(blocks * uint64(blockSize)), true
}
