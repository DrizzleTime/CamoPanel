package host

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// ErrUnreadable is returned when live host metrics cannot be read.
// Callers must not replace it with demo data.
var ErrUnreadable = errors.New("无法读取主机状态")

type linuxSampler struct {
	read   func(string) ([]byte, error)
	statfs func(string) (uint64, uint64, error)
	pids   func() ([]int, error)
	ticks  int
	sleep  func(time.Duration)
	hist   *history

	mu         sync.Mutex
	prevCPU    *cpuTimes
	prevNet    map[string]netCounters
	prevNetAt  time.Time
	prevProc   map[int]uint64
	prevProcAt time.Time
}

func newLinux() *linuxSampler {
	return &linuxSampler{
		read:     os.ReadFile,
		statfs:   statfsUsage,
		pids:     listPIDs,
		ticks:    clockTicks(),
		sleep:    time.Sleep,
		hist:     &history{},
		prevNet:  map[string]netCounters{},
		prevProc: map[int]uint64{},
	}
}

func clockTicks() int {
	// /proc/[pid]/stat counts time in USER_HZ. On Linux that value is 100.
	return 100
}

func statfsUsage(path string) (uint64, uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	if stat.Bsize <= 0 {
		return 0, 0, fmt.Errorf("无效的块大小")
	}
	bsize := uint64(stat.Bsize)
	total := stat.Blocks * bsize
	available := stat.Bavail * bsize
	if available > total {
		available = total
	}
	return total, total - available, nil
}

func listPIDs() ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	pids := make([]int, 0, len(entries))
	for _, entry := range entries {
		pid, convErr := strconv.Atoi(entry.Name())
		if convErr == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids, nil
}

func (s *linuxSampler) Overview(now time.Time) (Overview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now = now.UTC()

	cpu, err := s.sampleCPU()
	if err != nil {
		return Overview{}, ErrUnreadable
	}
	memRaw, err := s.read("/proc/meminfo")
	if err != nil {
		return Overview{}, ErrUnreadable
	}
	mem, swap, err := parseMeminfo(string(memRaw))
	if err != nil {
		return Overview{}, ErrUnreadable
	}
	loadRaw, err := s.read("/proc/loadavg")
	if err != nil {
		return Overview{}, ErrUnreadable
	}
	load, err := parseLoadavg(string(loadRaw))
	if err != nil {
		return Overview{}, ErrUnreadable
	}
	upRaw, err := s.read("/proc/uptime")
	if err != nil {
		return Overview{}, ErrUnreadable
	}
	uptime, err := parseUptime(string(upRaw))
	if err != nil {
		return Overview{}, ErrUnreadable
	}
	netRaw, err := s.read("/proc/net/dev")
	if err != nil {
		return Overview{}, ErrUnreadable
	}
	nets, err := s.sampleNets(string(netRaw), now)
	if err != nil {
		return Overview{}, ErrUnreadable
	}
	disks := s.sampleDisks()
	point := Point{At: stamped(now), CPUPercent: cpu.Percent, MemPercent: mem.Percent}
	return Overview{
		Mode:          "host",
		SampledAt:     stamped(now),
		UptimeSeconds: uptime,
		CPU:           cpu,
		Memory:        mem,
		Swap:          swap,
		Load:          load,
		Disks:         disks,
		Networks:      nets,
		History:       s.hist.add(point, now),
	}, nil
}

func (s *linuxSampler) sampleCPU() (CPU, error) {
	current, err := s.readCPU()
	if err != nil {
		return CPU{}, err
	}
	if s.prevCPU == nil {
		s.sleep(200 * time.Millisecond)
		next, readErr := s.readCPU()
		if readErr != nil {
			return CPU{}, readErr
		}
		percent := cpuPercent(current, next)
		s.prevCPU = &next
		return CPU{Percent: percent, Cores: next.cores}, nil
	}
	percent := cpuPercent(*s.prevCPU, current)
	cores := current.cores
	s.prevCPU = &current
	return CPU{Percent: percent, Cores: cores}, nil
}

func (s *linuxSampler) readCPU() (cpuTimes, error) {
	raw, err := s.read("/proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	return parseCPUStat(string(raw))
}

func (s *linuxSampler) sampleNets(data string, now time.Time) ([]Network, error) {
	counters, err := parseNetDev(data)
	if err != nil {
		return nil, err
	}
	elapsed := now.Sub(s.prevNetAt).Seconds()
	names := make([]string, 0, len(counters))
	for name := range counters {
		if keepNIC(name) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		for name := range counters {
			if name != "lo" {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	list := make([]Network, 0, len(names))
	for _, name := range names {
		cur := counters[name]
		item := Network{Name: name, RxBytes: cur.rx, TxBytes: cur.tx}
		if prev, ok := s.prevNet[name]; ok && elapsed > 0 && cur.rx >= prev.rx && cur.tx >= prev.tx {
			item.RxPerSec = round1(float64(cur.rx-prev.rx) / elapsed)
			item.TxPerSec = round1(float64(cur.tx-prev.tx) / elapsed)
		}
		list = append(list, item)
	}
	s.prevNet = counters
	s.prevNetAt = now
	return list, nil
}

func keepNIC(name string) bool {
	switch {
	case name == "lo" || name == "docker0":
		return false
	case strings.HasPrefix(name, "veth"), strings.HasPrefix(name, "br-"), strings.HasPrefix(name, "virbr"):
		return false
	default:
		return true
	}
}

func (s *linuxSampler) sampleDisks() []Disk {
	raw, err := s.read("/proc/mounts")
	if err != nil {
		return []Disk{}
	}
	entries := parseMounts(string(raw))
	sort.Slice(entries, func(i, j int) bool { return entries[i].mount < entries[j].mount })
	disks := make([]Disk, 0, len(entries))
	for _, entry := range entries {
		total, used, statErr := s.statfs(entry.mount)
		if statErr != nil || total == 0 {
			continue
		}
		disks = append(disks, Disk{
			Mount:   entry.mount,
			Device:  entry.device,
			Total:   total,
			Used:    used,
			Percent: round1(float64(used) / float64(total) * 100),
		})
	}
	return disks
}

func (s *linuxSampler) Processes(now time.Time, limit int, sortBy string) ([]Process, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now = now.UTC()
	memRaw, err := s.read("/proc/meminfo")
	if err != nil {
		return nil, ErrUnreadable
	}
	mem, _, err := parseMeminfo(string(memRaw))
	if err != nil {
		return nil, ErrUnreadable
	}
	pids, err := s.pids()
	if err != nil {
		return nil, ErrUnreadable
	}
	elapsed := now.Sub(s.prevProcAt).Seconds()
	nextTicks := make(map[int]uint64, len(pids))
	list := make([]Process, 0, len(pids))
	for _, pid := range pids {
		statRaw, statErr := s.read(fmt.Sprintf("/proc/%d/stat", pid))
		if statErr != nil {
			continue
		}
		comm, ticks, parseErr := parseProcStat(string(statRaw))
		if parseErr != nil {
			continue
		}
		nextTicks[pid] = ticks
		name := comm
		var rss uint64
		if statusRaw, statusErr := s.read(fmt.Sprintf("/proc/%d/status", pid)); statusErr == nil {
			statusName, statusRSS := parseStatus(string(statusRaw))
			if statusName != "" {
				name = statusName
			}
			rss = statusRSS
		}
		cpu := 0.0
		if prev, ok := s.prevProc[pid]; ok && elapsed > 0 && ticks >= prev && s.ticks > 0 {
			cpu = float64(ticks-prev) / (elapsed * float64(s.ticks)) * 100
		}
		memPercent := 0.0
		if mem.Total > 0 {
			memPercent = float64(rss) / float64(mem.Total) * 100
		}
		list = append(list, Process{
			PID:        pid,
			Name:       name,
			CPUPercent: round1(cpu),
			MemPercent: round1(memPercent),
			RSS:        rss,
		})
	}
	s.prevProc = nextTicks
	s.prevProcAt = now
	return limitProcesses(list, limit, sortBy), nil
}
