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
	Version, APIVersion, OperatingSystem, HostName string
	Healthy                                        bool
	CheckedAt                                      time.Time
	Diagnostic                                     string
	Workloads                                      targetCountMetric
	Images                                         targetImageMetric
	Volumes                                        targetImageMetric
	Networks                                       targetImageMetric
	CPU, Memory, Disk                              targetUsageMetric
}

type runtimeTargetSnapshotCollector interface {
	Collect(context.Context, store.Target) targetRuntimeSummary
}

type dockerTargetSnapshotCollector struct{}

type summaryCacheEntry struct {
	summary   targetRuntimeSummary
	expiresAt time.Time
}

// summaryCache 保存短时进程内摘要，并合并同一目标的并发采集请求。
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

// runtimeTargetCollector 按 runtime provider 选择摘要采集器；
// provider 为 "docker" 时返回 Docker 采集器，其它 provider 返回 nil。
func runtimeTargetCollector(provider string) runtimeTargetSnapshotCollector {
	if provider == "docker" {
		return dockerTargetSnapshotCollector{}
	}
	return nil
}

// collectTargetSummary 使用目标配置的 provider 采集 runtime 摘要；
// provider 没有注册采集器时返回标记为不可用的摘要。
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
	info, infoErr := client.Info(collectCtx, mobyclient.InfoOptions{})
	if infoErr == nil {
		result.OperatingSystem = info.Info.OperatingSystem
		result.HostName = info.Info.Name
	}
	result.Workloads = collectContainerMetric(collectCtx, client)
	result.Images = collectImageMetric(collectCtx, client)
	result.Volumes = collectVolumeMetric(collectCtx, client)
	result.Networks = collectNetworkMetric(collectCtx, client)
	result.CPU, result.Memory = collectHostUsage(collectCtx, collectHostCPUUsage, collectHostMemoryUsage)
	result.Disk = collectDockerFilesystemUsage(info, infoErr)
	return result
}

// closeDockerClient 关闭 Docker client。
func closeDockerClient(client *mobyclient.Client) {
	_ = client.Close()
}

// collectContainerMetric 采集 Docker workload 数量并识别活跃容器；
// Docker 无法提供容器列表时将指标标记为不可用。
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

// collectImageMetric 采集 Docker image 总数；Docker image 查询失败时将返回指标标记为不可用。
func collectImageMetric(ctx context.Context, client *mobyclient.Client) targetImageMetric {
	images, err := client.ImageList(ctx, mobyclient.ImageListOptions{All: true})
	if err != nil {
		return targetImageMetric{UnavailableReason: "Docker image metrics are unavailable"}
	}
	return targetImageMetric{Available: true, Total: int64(len(images.Items))}
}

// collectVolumeMetric 采集 Docker volume 总数；Docker 无法提供 volume 列表时将返回指标标记为不可用。
func collectVolumeMetric(ctx context.Context, client *mobyclient.Client) targetImageMetric {
	volumes, err := client.VolumeList(ctx, mobyclient.VolumeListOptions{})
	if err != nil {
		return targetImageMetric{UnavailableReason: "Docker volume metrics are unavailable"}
	}
	return targetImageMetric{Available: true, Total: int64(len(volumes.Items))}
}

// collectNetworkMetric 采集 Docker network 总数；无法读取 network 列表时将返回指标标记为不可用。
func collectNetworkMetric(ctx context.Context, client *mobyclient.Client) targetImageMetric {
	networks, err := client.NetworkList(ctx, mobyclient.NetworkListOptions{})
	if err != nil {
		return targetImageMetric{UnavailableReason: "Docker network metrics are unavailable"}
	}
	return targetImageMetric{Available: true, Total: int64(len(networks.Items))}
}

// collectDockerFilesystemUsage 采集 Docker 数据目录文件系统的使用指标；
// 无法获取 Docker 信息或文件系统统计时将指标标记为不可用。
func collectDockerFilesystemUsage(info mobyclient.SystemInfoResult, infoErr error) targetUsageMetric {
	var fs syscall.Statfs_t
	if infoErr == nil && info.Info.DockerRootDir != "" && syscall.Statfs(info.Info.DockerRootDir, &fs) == nil {
		return filesystemUsageMetric(fs.Blocks, fs.Bfree, fs.Bsize)
	}
	return targetUsageMetric{UnavailableReason: "Docker data directory filesystem is unavailable"}
}

type hostUsageCollector func(context.Context) targetUsageMetric

// collectHostUsage 采集 CPU 与内存使用指标。
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

// collectHostMemoryUsage 采集主机内存使用指标。
//
// 返回已用量、总量和使用率；无法采集内存快照或总量为零时将指标标记为不可用。
func collectHostMemoryUsage(ctx context.Context) targetUsageMetric {
	snapshot, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil || snapshot == nil || snapshot.Total == 0 {
		return targetUsageMetric{UnavailableReason: "Host memory metrics are unavailable"}
	}
	return targetUsageMetric{Available: true, UsedBytes: uint64ToInt64(snapshot.Used), TotalBytes: uint64ToInt64(snapshot.Total), UsagePercent: snapshot.UsedPercent}
}

// uint64ToInt64 将 uint64 转换为 int64，超过 int64 最大值时截断为最大值。
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

// filesystemBytes 在输入可产生有效 int64 时将块数和块大小转换为字节数；
// 块大小非正或结果溢出 int64 时返回 false。
func filesystemBytes(blocks uint64, blockSize int64) (int64, bool) {
	if blockSize <= 0 || blocks > uint64(math.MaxInt64)/uint64(blockSize) {
		return 0, false
	}
	//nolint:gosec // 上面的溢出检查已证明该乘积可安全转换为 int64。
	return int64(blocks * uint64(blockSize)), true
}
