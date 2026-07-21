package monitor

import (
	"context"
	"math"
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	gopsnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	generated "graft/server/internal/contract/openapi/generated"
)

type hostObservationCounters struct {
	networkObservedAt time.Time
	diskIOObservedAt  time.Time
	network           networkCounters
	diskIO            diskIOCounters
}

type networkCounters struct {
	bytesSent   uint64
	bytesRecv   uint64
	packetsSent uint64
	packetsRecv uint64
}

type diskIOCounters struct {
	readBytes   uint64
	writeBytes  uint64
	readCount   uint64
	writeCount  uint64
	readTimeMs  uint64
	writeTimeMs uint64
}

// collectHostObservability 收集补充的主机与当前进程观测值；无法取得的值保持 nil，避免将采集失败伪装为零。
func (p *Module) collectHostObservability(ctx context.Context) generated.ServerStatusHostObservability {
	result := generated.ServerStatusHostObservability{
		Network: generated.ServerStatusHostNetwork{},
		DiskIo:  generated.ServerStatusHostDiskIo{},
		Tcp:     collectTCPObservability(ctx),
		Process: collectProcessObservability(ctx),
	}

	network, networkAvailable := collectNetworkCounters(ctx)
	diskIO, diskAvailable := collectDiskIOCounters(ctx)
	if p == nil {
		return result
	}
	result.Network, result.DiskIo = p.collectHostRateObservability(network, networkAvailable, diskIO, diskAvailable)
	return result
}

func (p *Module) collectHostRateObservability(
	network networkCounters,
	networkAvailable bool,
	diskIO diskIOCounters,
	diskAvailable bool,
) (generated.ServerStatusHostNetwork, generated.ServerStatusHostDiskIo) {
	resultNetwork := generated.ServerStatusHostNetwork{}
	resultDiskIO := generated.ServerStatusHostDiskIo{}
	if !networkAvailable && !diskAvailable {
		return resultNetwork, resultDiskIO
	}

	p.hostObservabilityMu.Lock()
	defer p.hostObservabilityMu.Unlock()

	now := time.Now().UTC()
	previous := p.hostObservationCounters
	resultNetwork = networkRateObservability(previous, network, networkAvailable, now)
	resultDiskIO = diskIORateObservability(previous, diskIO, diskAvailable, now)
	p.hostObservationCounters = nextHostObservationCounters(previous, network, networkAvailable, diskIO, diskAvailable, now)

	return resultNetwork, resultDiskIO
}

func networkRateObservability(
	previous *hostObservationCounters,
	current networkCounters,
	available bool,
	now time.Time,
) generated.ServerStatusHostNetwork {
	if previous == nil || !available || !now.After(previous.networkObservedAt) {
		return generated.ServerStatusHostNetwork{}
	}
	return buildNetworkObservability(previous.network, current, now.Sub(previous.networkObservedAt))
}

func diskIORateObservability(
	previous *hostObservationCounters,
	current diskIOCounters,
	available bool,
	now time.Time,
) generated.ServerStatusHostDiskIo {
	if previous == nil || !available || !now.After(previous.diskIOObservedAt) {
		return generated.ServerStatusHostDiskIo{}
	}
	return buildDiskIOObservability(previous.diskIO, current, now.Sub(previous.diskIOObservedAt))
}

func nextHostObservationCounters(
	previous *hostObservationCounters,
	network networkCounters,
	networkAvailable bool,
	diskIO diskIOCounters,
	diskAvailable bool,
	now time.Time,
) *hostObservationCounters {
	// 每类计数器只在自身采集成功后才推进基线，避免失败样本污染下一次增量。
	next := hostObservationCounters{}
	if previous != nil {
		next = *previous
	}
	if networkAvailable {
		next.network = network
		next.networkObservedAt = now
	}
	if diskAvailable {
		next.diskIO = diskIO
		next.diskIOObservedAt = now
	}
	return &next
}

func collectNetworkCounters(ctx context.Context) (networkCounters, bool) {
	if ctx == nil {
		return networkCounters{}, false
	}
	interfaces, err := gopsnet.IOCountersWithContext(ctx, false)
	if err != nil {
		return networkCounters{}, false
	}
	var result networkCounters
	for _, item := range interfaces {
		result.bytesSent += item.BytesSent
		result.bytesRecv += item.BytesRecv
		result.packetsSent += item.PacketsSent
		result.packetsRecv += item.PacketsRecv
	}
	return result, true
}

func collectDiskIOCounters(ctx context.Context) (diskIOCounters, bool) {
	if ctx == nil {
		return diskIOCounters{}, false
	}
	drives, err := disk.IOCountersWithContext(ctx)
	if err != nil {
		return diskIOCounters{}, false
	}
	var result diskIOCounters
	for _, item := range drives {
		result.readBytes += item.ReadBytes
		result.writeBytes += item.WriteBytes
		result.readCount += item.ReadCount
		result.writeCount += item.WriteCount
		result.readTimeMs += item.ReadTime
		result.writeTimeMs += item.WriteTime
	}
	return result, true
}

func buildNetworkObservability(previous networkCounters, current networkCounters, elapsed time.Duration) generated.ServerStatusHostNetwork {
	return generated.ServerStatusHostNetwork{
		SentBytesPerSecond:       counterRate(previous.bytesSent, current.bytesSent, elapsed),
		ReceivedBytesPerSecond:   counterRate(previous.bytesRecv, current.bytesRecv, elapsed),
		SentPacketsPerSecond:     counterRate(previous.packetsSent, current.packetsSent, elapsed),
		ReceivedPacketsPerSecond: counterRate(previous.packetsRecv, current.packetsRecv, elapsed),
	}
}

func buildDiskIOObservability(previous diskIOCounters, current diskIOCounters, elapsed time.Duration) generated.ServerStatusHostDiskIo {
	readCount := counterDelta(previous.readCount, current.readCount)
	writeCount := counterDelta(previous.writeCount, current.writeCount)
	return generated.ServerStatusHostDiskIo{
		ReadBytesPerSecond:    counterRate(previous.readBytes, current.readBytes, elapsed),
		WriteBytesPerSecond:   counterRate(previous.writeBytes, current.writeBytes, elapsed),
		ReadIops:              counterRate(previous.readCount, current.readCount, elapsed),
		WriteIops:             counterRate(previous.writeCount, current.writeCount, elapsed),
		ReadAverageLatencyMs:  averageLatencyMilliseconds(counterDelta(previous.readTimeMs, current.readTimeMs), readCount),
		WriteAverageLatencyMs: averageLatencyMilliseconds(counterDelta(previous.writeTimeMs, current.writeTimeMs), writeCount),
	}
}

func counterDelta(previous uint64, current uint64) *uint64 {
	if current < previous {
		return nil
	}
	delta := current - previous
	return &delta
}

func counterRate(previous uint64, current uint64, elapsed time.Duration) *float32 {
	delta := counterDelta(previous, current)
	if delta == nil || elapsed <= 0 {
		return nil
	}
	rate := float64(*delta) / elapsed.Seconds()
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate > math.MaxFloat32 {
		return nil
	}
	converted := float32(rate)
	return &converted
}

func averageLatencyMilliseconds(totalMilliseconds *uint64, completedOperations *uint64) *float32 {
	if totalMilliseconds == nil || completedOperations == nil || *completedOperations == 0 {
		return nil
	}
	value := float64(*totalMilliseconds) / float64(*completedOperations)
	if math.IsNaN(value) || math.IsInf(value, 0) || value > math.MaxFloat32 {
		return nil
	}
	converted := float32(value)
	return &converted
}

func collectTCPObservability(ctx context.Context) generated.ServerStatusHostTcp {
	if ctx == nil {
		return generated.ServerStatusHostTcp{}
	}
	connections, err := gopsnet.ConnectionsWithContext(ctx, "tcp")
	if err != nil {
		return generated.ServerStatusHostTcp{}
	}
	total, established, timeWait, closeWait := int64(len(connections)), int64(0), int64(0), int64(0)
	for _, connection := range connections {
		switch strings.ToUpper(connection.Status) {
		case "ESTABLISHED":
			established++
		case "TIME_WAIT":
			timeWait++
		case "CLOSE_WAIT":
			closeWait++
		}
	}
	return generated.ServerStatusHostTcp{Total: &total, Established: &established, TimeWait: &timeWait, CloseWait: &closeWait}
}

func collectProcessObservability(ctx context.Context) generated.ServerStatusHostProcess {
	if ctx == nil {
		return generated.ServerStatusHostProcess{}
	}
	pid := os.Getpid()
	if pid > math.MaxInt32 {
		return generated.ServerStatusHostProcess{}
	}
	// 已确认 pid 位于 int32 范围内，gopsutil 的进程 API 仍固定使用 int32。
	current, err := process.NewProcessWithContext(ctx, int32(pid)) // #nosec G115
	if err != nil {
		return generated.ServerStatusHostProcess{}
	}
	result := generated.ServerStatusHostProcess{}
	if memory, memoryErr := current.MemoryInfoWithContext(ctx); memoryErr == nil && memory != nil && memory.RSS <= math.MaxInt64 {
		rss := int64(memory.RSS)
		result.RssBytes = &rss
	}
	if descriptors, descriptorErr := current.NumFDsWithContext(ctx); descriptorErr == nil {
		value := int64(descriptors)
		result.OpenFileDescriptors = &value
	}
	if threads, threadErr := current.NumThreadsWithContext(ctx); threadErr == nil {
		value := int64(threads)
		result.OsThreads = &value
	}
	return result
}
