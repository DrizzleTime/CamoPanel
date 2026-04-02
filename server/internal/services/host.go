package services

import (
	"context"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

const (
	defaultHostSampleInterval = 5 * time.Second
	defaultHostHistoryLimit   = 48
)

type TopProcess struct {
	PID     int32   `json:"pid"`
	Name    string  `json:"name"`
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	MemoryB uint64  `json:"memory_bytes"`
}

type HostSummary struct {
	Hostname     string       `json:"hostname"`
	OS           string       `json:"os"`
	Platform     string       `json:"platform"`
	Kernel       string       `json:"kernel"`
	Architecture string       `json:"architecture"`
	CPUCores     int          `json:"cpu_cores"`
	CPUPercent   float64      `json:"cpu_percent"`
	Load1        float64      `json:"load_1"`
	Load5        float64      `json:"load_5"`
	MemoryUsed   uint64       `json:"memory_used"`
	MemoryTotal  uint64       `json:"memory_total"`
	DiskUsed     uint64       `json:"disk_used"`
	DiskTotal    uint64       `json:"disk_total"`
	TopCPU       []TopProcess `json:"top_cpu"`
	TopMemory    []TopProcess `json:"top_memory"`
	SampledAt    time.Time    `json:"sampled_at"`
}

type HostMetricsPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	CPUPercent    float64   `json:"cpu_percent"`
	Load1         float64   `json:"load_1"`
	Load5         float64   `json:"load_5"`
	MemoryUsed    uint64    `json:"memory_used"`
	MemoryTotal   uint64    `json:"memory_total"`
	DiskUsed      uint64    `json:"disk_used"`
	DiskTotal     uint64    `json:"disk_total"`
	NetworkRxRate float64   `json:"network_rx_rate"`
	NetworkTxRate float64   `json:"network_tx_rate"`
	DiskReadRate  float64   `json:"disk_read_rate"`
	DiskWriteRate float64   `json:"disk_write_rate"`
}

type HostMetrics struct {
	Summary               HostSummary        `json:"summary"`
	History               []HostMetricsPoint `json:"history"`
	SampleIntervalSeconds int                `json:"sample_interval_seconds"`
}

type hostCounterSnapshot struct {
	timestamp time.Time
	cpuTimes  cpu.TimesStat
	netRx     uint64
	netTx     uint64
	diskRead  uint64
	diskWrite uint64
}

type HostService struct {
	diskPath       string
	sampleInterval time.Duration
	maxSamples     int

	mu           sync.RWMutex
	summary      HostSummary
	history      []HostMetricsPoint
	lastCounters *hostCounterSnapshot
	diskDevice   string
}

func NewHostService(diskPath string) *HostService {
	if abs, err := filepath.Abs(diskPath); err == nil {
		diskPath = abs
	}

	service := &HostService{
		diskPath:       diskPath,
		sampleInterval: defaultHostSampleInterval,
		maxSamples:     defaultHostHistoryLimit,
	}

	_ = service.refresh(context.Background())
	go service.loop()

	return service
}

func (h *HostService) Summary(ctx context.Context) (HostSummary, error) {
	if err := h.ensureFresh(ctx); err != nil {
		return HostSummary{}, err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.summary, nil
}

func (h *HostService) Metrics(ctx context.Context) (HostMetrics, error) {
	if err := h.ensureFresh(ctx); err != nil {
		return HostMetrics{}, err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	history := append([]HostMetricsPoint(nil), h.history...)
	return HostMetrics{
		Summary:               h.summary,
		History:               history,
		SampleIntervalSeconds: int(h.sampleInterval / time.Second),
	}, nil
}

func (h *HostService) loop() {
	ticker := time.NewTicker(h.sampleInterval)
	defer ticker.Stop()

	for range ticker.C {
		_ = h.refresh(context.Background())
	}
}

func (h *HostService) ensureFresh(ctx context.Context) error {
	h.mu.RLock()
	stale := h.summary.SampledAt.IsZero() || time.Since(h.summary.SampledAt) > h.sampleInterval*2
	h.mu.RUnlock()

	if stale {
		return h.refresh(ctx)
	}
	return nil
}

func (h *HostService) refresh(ctx context.Context) error {
	summary, cpuTimes, netRx, netTx, diskRead, diskWrite, sampledAt, err := h.collectSnapshot(ctx)
	if err != nil {
		return err
	}

	point := HostMetricsPoint{
		Timestamp:   sampledAt,
		Load1:       summary.Load1,
		Load5:       summary.Load5,
		MemoryUsed:  summary.MemoryUsed,
		MemoryTotal: summary.MemoryTotal,
		DiskUsed:    summary.DiskUsed,
		DiskTotal:   summary.DiskTotal,
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.lastCounters != nil {
		seconds := sampledAt.Sub(h.lastCounters.timestamp).Seconds()
		if seconds > 0 {
			point.CPUPercent = cpuUsagePercent(h.lastCounters.cpuTimes, cpuTimes)
			point.NetworkRxRate = counterRate(h.lastCounters.netRx, netRx, seconds)
			point.NetworkTxRate = counterRate(h.lastCounters.netTx, netTx, seconds)
			point.DiskReadRate = counterRate(h.lastCounters.diskRead, diskRead, seconds)
			point.DiskWriteRate = counterRate(h.lastCounters.diskWrite, diskWrite, seconds)
		}
	}

	summary.CPUPercent = point.CPUPercent
	summary.SampledAt = sampledAt
	summary.TopCPU, summary.TopMemory = collectTopProcesses(ctx)

	h.summary = summary
	h.history = append(h.history, point)
	if extra := len(h.history) - h.maxSamples; extra > 0 {
		h.history = append([]HostMetricsPoint(nil), h.history[extra:]...)
	}
	h.lastCounters = &hostCounterSnapshot{
		timestamp: sampledAt,
		cpuTimes:  cpuTimes,
		netRx:     netRx,
		netTx:     netTx,
		diskRead:  diskRead,
		diskWrite: diskWrite,
	}

	return nil
}

func (h *HostService) collectSnapshot(ctx context.Context) (HostSummary, cpu.TimesStat, uint64, uint64, uint64, uint64, time.Time, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return HostSummary{}, cpu.TimesStat{}, 0, 0, 0, 0, time.Time{}, err
	}

	loadInfo, err := load.Avg()
	if err != nil {
		return HostSummary{}, cpu.TimesStat{}, 0, 0, 0, 0, time.Time{}, err
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return HostSummary{}, cpu.TimesStat{}, 0, 0, 0, 0, time.Time{}, err
	}

	diskInfo, err := disk.Usage(h.diskPath)
	if err != nil {
		return HostSummary{}, cpu.TimesStat{}, 0, 0, 0, 0, time.Time{}, err
	}

	cores, err := cpu.Counts(true)
	if err != nil {
		cores = runtime.NumCPU()
	}

	cpuTimes, err := cpu.TimesWithContext(ctx, false)
	if err != nil || len(cpuTimes) == 0 {
		return HostSummary{}, cpu.TimesStat{}, 0, 0, 0, 0, time.Time{}, err
	}

	var netRx uint64
	var netTx uint64
	if counters, counterErr := gnet.IOCountersWithContext(ctx, false); counterErr == nil && len(counters) > 0 {
		netRx = counters[0].BytesRecv
		netTx = counters[0].BytesSent
	}

	diskRead, diskWrite, _ := h.readDiskCounters(ctx)
	sampledAt := time.Now().UTC()

	return HostSummary{
		Hostname:     hostInfo.Hostname,
		OS:           hostInfo.OS,
		Platform:     hostInfo.Platform + " " + hostInfo.PlatformVersion,
		Kernel:       hostInfo.KernelVersion,
		Architecture: runtime.GOARCH,
		CPUCores:     cores,
		Load1:        loadInfo.Load1,
		Load5:        loadInfo.Load5,
		MemoryUsed:   memInfo.Used,
		MemoryTotal:  memInfo.Total,
		DiskUsed:     diskInfo.Used,
		DiskTotal:    diskInfo.Total,
	}, cpuTimes[0], netRx, netTx, diskRead, diskWrite, sampledAt, nil
}

func (h *HostService) readDiskCounters(ctx context.Context) (uint64, uint64, error) {
	device := h.resolveDiskDevice(ctx)
	if device != "" {
		counters, err := disk.IOCountersWithContext(ctx, device)
		if err == nil {
			if stat, ok := counters[device]; ok {
				return stat.ReadBytes, stat.WriteBytes, nil
			}
		}
	}

	counters, err := disk.IOCountersWithContext(ctx)
	if err != nil {
		return 0, 0, err
	}

	var readBytes uint64
	var writeBytes uint64
	for _, stat := range counters {
		readBytes += stat.ReadBytes
		writeBytes += stat.WriteBytes
	}
	return readBytes, writeBytes, nil
}

func (h *HostService) resolveDiskDevice(ctx context.Context) string {
	h.mu.RLock()
	if h.diskDevice != "" {
		defer h.mu.RUnlock()
		return h.diskDevice
	}
	h.mu.RUnlock()

	partitions, err := disk.PartitionsWithContext(ctx, true)
	if err != nil {
		return ""
	}

	bestMount := ""
	bestDevice := ""
	for _, partition := range partitions {
		if !pathOnMount(h.diskPath, partition.Mountpoint) {
			continue
		}
		if len(partition.Mountpoint) < len(bestMount) {
			continue
		}

		device := normalizeDiskDevice(partition.Device)
		if device == "" {
			continue
		}

		bestMount = partition.Mountpoint
		bestDevice = device
	}

	if bestDevice == "" {
		return ""
	}

	h.mu.Lock()
	if h.diskDevice == "" {
		h.diskDevice = bestDevice
	}
	h.mu.Unlock()

	return bestDevice
}

func pathOnMount(targetPath string, mountPoint string) bool {
	targetPath = filepath.Clean(targetPath)
	mountPoint = filepath.Clean(mountPoint)

	if mountPoint == "/" {
		return true
	}
	return targetPath == mountPoint || strings.HasPrefix(targetPath, mountPoint+string(filepath.Separator))
}

func normalizeDiskDevice(device string) string {
	device = strings.TrimSpace(device)
	if device == "" {
		return ""
	}
	if strings.HasPrefix(device, "/dev/") {
		return filepath.Base(device)
	}
	return device
}

func cpuUsagePercent(previous cpu.TimesStat, current cpu.TimesStat) float64 {
	previousTotal := previous.Total()
	currentTotal := current.Total()
	if runtime.GOOS == "linux" {
		previousTotal -= previous.Guest + previous.GuestNice
		currentTotal -= current.Guest + current.GuestNice
	}

	previousBusy := previousTotal - previous.Idle - previous.Iowait
	currentBusy := currentTotal - current.Idle - current.Iowait
	if currentTotal <= previousTotal || currentBusy <= previousBusy {
		return 0
	}

	percent := (currentBusy - previousBusy) / (currentTotal - previousTotal) * 100
	switch {
	case percent < 0:
		return 0
	case percent > 100:
		return 100
	default:
		return percent
	}
}

func counterRate(previous uint64, current uint64, seconds float64) float64 {
	if seconds <= 0 || current < previous {
		return 0
	}
	return float64(current-previous) / seconds
}

const topProcessCount = 5

func collectTopProcesses(ctx context.Context) (topCPU []TopProcess, topMem []TopProcess) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil || len(procs) == 0 {
		return nil, nil
	}

	type procInfo struct {
		pid     int32
		name    string
		cpuPct  float64
		memPct  float64
		memRSS  uint64
	}

	infos := make([]procInfo, 0, len(procs))
	for _, p := range procs {
		name, nameErr := p.NameWithContext(ctx)
		if nameErr != nil || name == "" {
			continue
		}

		cpuPct, cpuErr := p.CPUPercentWithContext(ctx)
		memInfo, memErr := p.MemoryInfoWithContext(ctx)
		memPct, memPctErr := p.MemoryPercentWithContext(ctx)

		var rss uint64
		if memErr == nil && memInfo != nil {
			rss = memInfo.RSS
		}

		var mPct float64
		if memPctErr == nil {
			mPct = float64(memPct)
		}

		var cPct float64
		if cpuErr == nil {
			cPct = cpuPct
		}

		if cPct <= 0 && rss == 0 {
			continue
		}

		infos = append(infos, procInfo{
			pid:    p.Pid,
			name:   name,
			cpuPct: cPct,
			memPct: mPct,
			memRSS: rss,
		})
	}

	byCPU := make([]procInfo, len(infos))
	copy(byCPU, infos)
	sort.Slice(byCPU, func(i, j int) bool { return byCPU[i].cpuPct > byCPU[j].cpuPct })

	byMem := make([]procInfo, len(infos))
	copy(byMem, infos)
	sort.Slice(byMem, func(i, j int) bool { return byMem[i].memRSS > byMem[j].memRSS })

	toSlice := func(src []procInfo) []TopProcess {
		n := topProcessCount
		if len(src) < n {
			n = len(src)
		}
		out := make([]TopProcess, n)
		for i := 0; i < n; i++ {
			out[i] = TopProcess{
				PID:     src[i].pid,
				Name:    src[i].name,
				CPU:     src[i].cpuPct,
				Memory:  src[i].memPct,
				MemoryB: src[i].memRSS,
			}
		}
		return out
	}

	return toSlice(byCPU), toSlice(byMem)
}
