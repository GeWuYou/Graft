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
	networkObservedAt    time.Time
	diskIOObservedAt     time.Time
	network              networkCounters
	diskIO               diskIOCounters
	networkObservability generated.ServerStatusHostNetwork
	diskIOObservability  generated.ServerStatusHostDiskIo
}

type networkCounters struct {
	bytesSent   uint64
	bytesRecv   uint64
	packetsSent uint64
	packetsRecv uint64
}

type diskIOCounters struct {
	drives map[string]diskIOCounter
}

type diskIOCounter struct {
	readBytes   uint64
	writeBytes  uint64
	readCount   uint64
	writeCount  uint64
	readTimeMs  uint64
	writeTimeMs uint64
}

type hostObservationSample struct {
	network              networkCounters
	networkAvailable     bool
	diskIO               diskIOCounters
	diskIOAvailable      bool
	networkObservability generated.ServerStatusHostNetwork
	diskIOObservability  generated.ServerStatusHostDiskIo
	observedAt           time.Time
}

// sampleHostObservability 由固定间隔采样器推进主机计数器基线，避免请求频率改变趋势的统计区间。
func (p *Module) sampleHostObservability(ctx context.Context) generated.ServerStatusHostObservability {
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
	result.Network, result.DiskIo = p.sampleHostRateObservability(network, networkAvailable, diskIO, diskAvailable, time.Now().UTC())
	return result
}

// currentHostObservability 组合请求时的瞬时 TCP/进程指标和最近一次固定间隔采样得到的速率指标。
func (p *Module) currentHostObservability(ctx context.Context) generated.ServerStatusHostObservability {
	result := generated.ServerStatusHostObservability{
		Network: generated.ServerStatusHostNetwork{},
		DiskIo:  generated.ServerStatusHostDiskIo{},
		Tcp:     collectTCPObservability(ctx),
		Process: collectProcessObservability(ctx),
	}
	if p == nil {
		return result
	}
	result.Network, result.DiskIo = p.latestHostRateObservability()
	return result
}

func (p *Module) sampleHostRateObservability(
	network networkCounters,
	networkAvailable bool,
	diskIO diskIOCounters,
	diskAvailable bool,
	now time.Time,
) (generated.ServerStatusHostNetwork, generated.ServerStatusHostDiskIo) {
	resultNetwork := generated.ServerStatusHostNetwork{}
	resultDiskIO := generated.ServerStatusHostDiskIo{}
	if !networkAvailable && !diskAvailable {
		return resultNetwork, resultDiskIO
	}

	p.hostObservabilityMu.Lock()
	defer p.hostObservabilityMu.Unlock()

	previous := p.hostObservationCounters
	resultNetwork = networkRateObservability(previous, network, networkAvailable, now)
	resultDiskIO = diskIORateObservability(previous, diskIO, diskAvailable, now)
	p.hostObservationCounters = nextHostObservationCounters(previous, hostObservationSample{
		network:              network,
		networkAvailable:     networkAvailable,
		diskIO:               diskIO,
		diskIOAvailable:      diskAvailable,
		networkObservability: resultNetwork,
		diskIOObservability:  resultDiskIO,
		observedAt:           now,
	})

	return resultNetwork, resultDiskIO
}

func (p *Module) latestHostRateObservability() (generated.ServerStatusHostNetwork, generated.ServerStatusHostDiskIo) {
	p.hostObservabilityMu.Lock()
	defer p.hostObservabilityMu.Unlock()

	if p.hostObservationCounters == nil {
		return generated.ServerStatusHostNetwork{}, generated.ServerStatusHostDiskIo{}
	}
	return p.hostObservationCounters.networkObservability, p.hostObservationCounters.diskIOObservability
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
	sample hostObservationSample,
) *hostObservationCounters {
	// 每类计数器只在自身采集成功后才推进基线，避免失败样本污染下一次增量。
	next := hostObservationCounters{}
	if previous != nil {
		next = *previous
	}
	if sample.networkAvailable {
		next.network = sample.network
		next.networkObservedAt = sample.observedAt
		next.networkObservability = sample.networkObservability
	}
	if sample.diskIOAvailable {
		next.diskIO = sample.diskIO
		next.diskIOObservedAt = sample.observedAt
		next.diskIOObservability = sample.diskIOObservability
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
	result := diskIOCounters{drives: make(map[string]diskIOCounter, len(drives))}
	for name, item := range drives {
		result.drives[name] = diskIOCounter{
			readBytes: item.ReadBytes, writeBytes: item.WriteBytes, readCount: item.ReadCount,
			writeCount: item.WriteCount, readTimeMs: item.ReadTime, writeTimeMs: item.WriteTime,
		}
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
	previousTotals, currentTotals, comparable := comparableDiskIOCounters(previous, current)
	if !comparable {
		return generated.ServerStatusHostDiskIo{}
	}
	readCount := counterDelta(previousTotals.readCount, currentTotals.readCount)
	writeCount := counterDelta(previousTotals.writeCount, currentTotals.writeCount)
	return generated.ServerStatusHostDiskIo{
		ReadBytesPerSecond:    counterRate(previousTotals.readBytes, currentTotals.readBytes, elapsed),
		WriteBytesPerSecond:   counterRate(previousTotals.writeBytes, currentTotals.writeBytes, elapsed),
		ReadIops:              counterRate(previousTotals.readCount, currentTotals.readCount, elapsed),
		WriteIops:             counterRate(previousTotals.writeCount, currentTotals.writeCount, elapsed),
		ReadAverageLatencyMs:  averageLatencyMilliseconds(counterDelta(previousTotals.readTimeMs, currentTotals.readTimeMs), readCount),
		WriteAverageLatencyMs: averageLatencyMilliseconds(counterDelta(previousTotals.writeTimeMs, currentTotals.writeTimeMs), writeCount),
	}
}

func comparableDiskIOCounters(previous diskIOCounters, current diskIOCounters) (diskIOCounter, diskIOCounter, bool) {
	var previousTotals, currentTotals diskIOCounter
	comparable := false
	for name, currentDrive := range current.drives {
		previousDrive, ok := previous.drives[name]
		if !ok {
			continue
		}
		comparable = true
		previousTotals.readBytes += previousDrive.readBytes
		previousTotals.writeBytes += previousDrive.writeBytes
		previousTotals.readCount += previousDrive.readCount
		previousTotals.writeCount += previousDrive.writeCount
		previousTotals.readTimeMs += previousDrive.readTimeMs
		previousTotals.writeTimeMs += previousDrive.writeTimeMs
		currentTotals.readBytes += currentDrive.readBytes
		currentTotals.writeBytes += currentDrive.writeBytes
		currentTotals.readCount += currentDrive.readCount
		currentTotals.writeCount += currentDrive.writeCount
		currentTotals.readTimeMs += currentDrive.readTimeMs
		currentTotals.writeTimeMs += currentDrive.writeTimeMs
	}
	return previousTotals, currentTotals, comparable
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
