package host

import (
	"math"
	"slices"
	"time"
)

type demoSampler struct {
	hist *history
}

func newDemo() *demoSampler {
	sampler := &demoSampler{hist: &history{}}
	now := time.Now().UTC()
	for i := 60; i >= 1; i-- {
		at := now.Add(-time.Duration(i) * time.Minute)
		sampler.hist.add(Point{At: stamped(at), CPUPercent: wave(at, 18, 8), MemPercent: wave(at, 46, 4)}, at)
	}
	return sampler
}

func wave(at time.Time, base, amp float64) float64 {
	phase := float64(at.Unix()%3600) / 3600 * 2 * math.Pi
	return round1(base + amp*math.Sin(phase))
}

func (d *demoSampler) Overview(now time.Time) (Overview, error) {
	now = now.UTC()
	cpu := wave(now, 18, 8)
	memPercent := wave(now, 46, 4)
	const memTotal = 8 * 1024 * 1024 * 1024
	memUsed := uint64(float64(memTotal) * memPercent / 100)
	const swapTotal = 2 * 1024 * 1024 * 1024
	swapPercent := wave(now, 6, 3)
	swapUsed := uint64(float64(swapTotal) * swapPercent / 100)
	point := Point{At: stamped(now), CPUPercent: cpu, MemPercent: memPercent}
	overview := Overview{
		Mode:          "demo",
		SampledAt:     stamped(now),
		UptimeSeconds: 345678,
		CPU:           CPU{Percent: cpu, Cores: 4},
		Memory:        Memory{Total: memTotal, Used: memUsed, Available: memTotal - memUsed, Percent: memPercent},
		Swap:          Memory{Total: swapTotal, Used: swapUsed, Available: swapTotal - swapUsed, Percent: swapPercent},
		Load:          Load{Load1: wave(now, 0.4, 0.2), Load5: wave(now, 0.5, 0.15), Load15: wave(now, 0.45, 0.1)},
		Disks: []Disk{
			{Mount: "/", Device: "/dev/vda1", Total: 80 * 1024 * 1024 * 1024, Used: 28 * 1024 * 1024 * 1024, Percent: 35},
			{Mount: "/data", Device: "/dev/vdb1", Total: 200 * 1024 * 1024 * 1024, Used: 40 * 1024 * 1024 * 1024, Percent: 20},
		},
		Networks: []Network{
			{Name: "eth0", RxBytes: 120_000_000, TxBytes: 48_000_000, RxPerSec: wave(now, 180000, 40000), TxPerSec: wave(now, 42000, 8000)},
		},
		History: d.hist.add(point, now),
	}
	return overview, nil
}

func (d *demoSampler) Processes(_ time.Time, limit int, sortBy string) ([]Process, error) {
	list := []Process{
		{PID: 1, Name: "systemd", CPUPercent: 0.2, MemPercent: 0.4, RSS: 12 * 1024 * 1024},
		{PID: 412, Name: "dockerd", CPUPercent: 3.4, MemPercent: 4.8, RSS: 180 * 1024 * 1024},
		{PID: 880, Name: "containerd", CPUPercent: 1.1, MemPercent: 2.2, RSS: 96 * 1024 * 1024},
		{PID: 1102, Name: "nginx", CPUPercent: 0.8, MemPercent: 1.1, RSS: 28 * 1024 * 1024},
		{PID: 1330, Name: "camopanel", CPUPercent: 2.6, MemPercent: 3.5, RSS: 140 * 1024 * 1024},
		{PID: 1504, Name: "postgres", CPUPercent: 6.2, MemPercent: 11.4, RSS: 520 * 1024 * 1024},
		{PID: 1620, Name: "redis-server", CPUPercent: 1.4, MemPercent: 2.8, RSS: 86 * 1024 * 1024},
		{PID: 1744, Name: "node", CPUPercent: 4.1, MemPercent: 5.6, RSS: 240 * 1024 * 1024},
		{PID: 1888, Name: "python", CPUPercent: 0.6, MemPercent: 1.9, RSS: 64 * 1024 * 1024},
		{PID: 2011, Name: "sshd", CPUPercent: 0.1, MemPercent: 0.3, RSS: 8 * 1024 * 1024},
		{PID: 2200, Name: "journald", CPUPercent: 0.3, MemPercent: 0.7, RSS: 18 * 1024 * 1024},
		{PID: 2408, Name: "rsyslogd", CPUPercent: 0.2, MemPercent: 0.5, RSS: 10 * 1024 * 1024},
	}
	return limitProcesses(list, limit, sortBy), nil
}

func limitProcesses(list []Process, limit int, sortBy string) []Process {
	slices.SortStableFunc(list, func(a, b Process) int {
		av, bv := a.CPUPercent, b.CPUPercent
		if sortBy == "mem" {
			av, bv = a.MemPercent, b.MemPercent
		}
		switch {
		case av > bv:
			return -1
		case av < bv:
			return 1
		case a.PID < b.PID:
			return -1
		case a.PID > b.PID:
			return 1
		default:
			return 0
		}
	})
	if limit > len(list) {
		limit = len(list)
	}
	if limit < 0 {
		limit = 0
	}
	out := make([]Process, limit)
	copy(out, list[:limit])
	return out
}
