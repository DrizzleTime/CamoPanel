package host

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestParsers(t *testing.T) {
	cpu, err := parseCPUStat("cpu  100 0 0 100 0 0 0 0\ncpu0 100 0 0 100 0 0 0 0\n")
	if err != nil || cpu.cores != 1 || cpu.total == 0 {
		t.Fatalf("cpu=%+v err=%v", cpu, err)
	}
	next, err := parseCPUStat("cpu  150 0 0 120 0 0 0 0\ncpu0 150 0 0 120 0 0 0 0\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := cpuPercent(cpu, next); got != 71.4 {
		t.Fatalf("cpu percent=%v", got)
	}
	mem, swap, err := parseMeminfo("MemTotal: 8192 kB\nMemAvailable: 4096 kB\nSwapTotal: 1024 kB\nSwapFree: 512 kB\n")
	if err != nil || mem.Percent != 50 || swap.Used != 512*1024 {
		t.Fatalf("mem=%+v swap=%+v err=%v", mem, swap, err)
	}
	load, err := parseLoadavg("0.50 0.40 0.30 1/20 9\n")
	if err != nil || load.Load1 != 0.5 || load.Load15 != 0.3 {
		t.Fatalf("load=%+v err=%v", load, err)
	}
	mounts := parseMounts("/dev/vda1 / ext4 rw 0 0\ntmpfs /run tmpfs rw 0 0\n/dev/vda1 /var/lib\\040data ext4 rw 0 0\n/dev/vdb /data ext4 rw 0 0\n")
	if len(mounts) != 2 {
		t.Fatalf("mounts=%+v", mounts)
	}
	for _, entry := range mounts {
		if entry.device == "/dev/vda1" && entry.mount != "/" {
			t.Fatalf("kept longer mount: %+v", entry)
		}
	}
	name, ticks, err := parseProcStat("12 (web server) S 0 0 0 0 0 0 0 0 0 0 10 20\n")
	if err != nil || name != "web server" || ticks != 30 {
		t.Fatalf("name=%s ticks=%d err=%v", name, ticks, err)
	}
	statusName, rss := parseStatus("Name:\tweb-server\nVmRSS:\t    2048 kB\n")
	if statusName != "web-server" || rss != 2048*1024 {
		t.Fatalf("status name=%s rss=%d", statusName, rss)
	}
}

func TestLinuxSamplerReadsInjectedProc(t *testing.T) {
	files := &scriptedProc{
		stat: []string{
			"cpu  100 0 0 100 0 0 0 0\ncpu0 100 0 0 100 0 0 0 0\n",
			"cpu  150 0 0 120 0 0 0 0\ncpu0 150 0 0 120 0 0 0 0\n",
		},
		mem:    "MemTotal: 8192 kB\nMemAvailable: 4096 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n",
		load:   "0.50 0.40 0.30 1/20 9\n",
		uptime: "100.5 10\n",
		nets: []string{
			"Inter-| Receive | Transmit\n face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed\n lo: 10 0 0 0 0 0 0 0 10 0 0 0 0 0 0 0\n eth0: 1000 0 0 0 0 0 0 0 500 0 0 0 0 0 0 0\n vethabc: 9 0 0 0 0 0 0 0 9 0 0 0 0 0 0 0\n",
			"Inter-| Receive | Transmit\n face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed\n lo: 10 0 0 0 0 0 0 0 10 0 0 0 0 0 0 0\n eth0: 3000 0 0 0 0 0 0 0 1500 0 0 0 0 0 0 0\n vethabc: 9 0 0 0 0 0 0 0 9 0 0 0 0 0 0 0\n",
		},
		mounts: "/dev/vda1 / ext4 rw 0 0\ntmpfs /run tmpfs rw 0 0\n/dev/vdb /data ext4 rw 0 0\n",
		procs: map[string][]string{
			"/proc/12/stat":   {"12 (web server) S 0 0 0 0 0 0 0 0 0 0 10 20\n", "12 (web server) S 0 0 0 0 0 0 0 0 0 0 15 25\n"},
			"/proc/12/status": {"Name:\tweb-server\nVmRSS:\t    2048 kB\n"},
			"/proc/13/stat":   {"13 (cache) S 0 0 0 0 0 0 0 0 0 0 1 1\n"},
			"/proc/13/status": {"Name:\tcache\nVmRSS:\t    4096 kB\n"},
		},
	}
	sampler := &linuxSampler{
		read: files.read,
		statfs: func(string) (uint64, uint64, error) {
			return 1000, 250, nil
		},
		pids:  func() ([]int, error) { return []int{12, 13}, nil },
		ticks: 100,
		sleep: func(time.Duration) {},
		hist:  &history{},
	}
	now := time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC)
	overview, err := sampler.Overview(now)
	if err != nil {
		t.Fatal(err)
	}
	if overview.Mode != "host" || overview.CPU.Percent != 71.4 || overview.CPU.Cores != 1 {
		t.Fatalf("cpu=%+v", overview.CPU)
	}
	if overview.Memory.Percent != 50 || len(overview.Disks) != 2 || overview.Disks[0].Percent != 25 {
		t.Fatalf("mem=%+v disks=%+v", overview.Memory, overview.Disks)
	}
	if len(overview.Networks) != 1 || overview.Networks[0].Name != "eth0" || overview.Networks[0].RxPerSec != 0 {
		t.Fatalf("nets=%+v", overview.Networks)
	}
	second, err := sampler.Overview(now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if second.Networks[0].RxPerSec != 2000 || second.Networks[0].TxPerSec != 1000 {
		t.Fatalf("rates=%+v", second.Networks[0])
	}
	if len(second.History) != 2 {
		t.Fatalf("history=%d", len(second.History))
	}

	firstProcs, err := sampler.Processes(now, 10, "cpu")
	if err != nil || len(firstProcs) != 2 || firstProcs[0].CPUPercent != 0 {
		t.Fatalf("first procs=%+v err=%v", firstProcs, err)
	}
	ranked, err := sampler.Processes(now.Add(time.Second), 1, "cpu")
	if err != nil || len(ranked) != 1 || ranked[0].Name != "web-server" || ranked[0].CPUPercent != 10 {
		t.Fatalf("ranked=%+v err=%v", ranked, err)
	}
	byMem, err := sampler.Processes(now.Add(2*time.Second), 1, "mem")
	if err != nil || byMem[0].Name != "cache" || byMem[0].MemPercent != 50 {
		t.Fatalf("by mem=%+v err=%v", byMem, err)
	}
}

func TestLinuxSamplerReportsReadFailure(t *testing.T) {
	sampler := &linuxSampler{
		read:  func(string) ([]byte, error) { return nil, os.ErrNotExist },
		sleep: func(time.Duration) {},
		hist:  &history{},
	}
	_, err := sampler.Overview(time.Now())
	if !errors.Is(err, ErrUnreadable) {
		t.Fatalf("err=%v", err)
	}
}

func TestLiveHostOverview(t *testing.T) {
	if _, err := os.Stat("/proc/stat"); err != nil {
		t.Skip("no /proc")
	}
	sampler := newLinux()
	overview, err := sampler.Overview(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if overview.Mode != "host" || overview.CPU.Cores < 1 || overview.Memory.Total == 0 || overview.UptimeSeconds <= 0 {
		t.Fatalf("%+v", overview)
	}
	processes, err := sampler.Processes(time.Now(), 5, "mem")
	if err != nil {
		t.Fatal(err)
	}
	if len(processes) == 0 || len(processes) > 5 {
		t.Fatalf("processes=%d", len(processes))
	}
}

func TestDemoOverviewIsStable(t *testing.T) {
	sampler := newDemo()
	now := time.Now().UTC()
	overview, err := sampler.Overview(now)
	if err != nil {
		t.Fatal(err)
	}
	if overview.Mode != "demo" || overview.CPU.Cores != 4 || len(overview.Disks) != 2 || len(overview.History) < 60 {
		t.Fatalf("mode=%s cores=%d disks=%d history=%d", overview.Mode, overview.CPU.Cores, len(overview.Disks), len(overview.History))
	}
	if overview.CPU.Percent < 0 || overview.CPU.Percent > 100 || overview.Memory.Percent < 0 || overview.Memory.Percent > 100 {
		t.Fatalf("percent cpu=%v mem=%v", overview.CPU.Percent, overview.Memory.Percent)
	}
	processes, err := sampler.Processes(now, 10, "cpu")
	if err != nil || len(processes) != 10 || processes[0].Name != "postgres" {
		t.Fatalf("%+v err=%v", processes, err)
	}
	byMem, err := sampler.Processes(now, 1, "mem")
	if err != nil || byMem[0].Name != "postgres" {
		t.Fatalf("%+v", byMem)
	}
}

type scriptedProc struct {
	stat   []string
	statN  int
	mem    string
	load   string
	uptime string
	nets   []string
	netN   int
	mounts string
	procs  map[string][]string
}

func (s *scriptedProc) read(path string) ([]byte, error) {
	switch path {
	case "/proc/stat":
		item := s.take(&s.stat, &s.statN)
		if item == "" {
			return nil, os.ErrNotExist
		}
		return []byte(item), nil
	case "/proc/meminfo":
		return []byte(s.mem), nil
	case "/proc/loadavg":
		return []byte(s.load), nil
	case "/proc/uptime":
		return []byte(s.uptime), nil
	case "/proc/net/dev":
		item := s.take(&s.nets, &s.netN)
		if item == "" {
			return nil, os.ErrNotExist
		}
		return []byte(item), nil
	case "/proc/mounts":
		return []byte(s.mounts), nil
	default:
		queue := s.procs[path]
		if len(queue) == 0 {
			return nil, os.ErrNotExist
		}
		item := queue[0]
		if len(queue) > 1 {
			s.procs[path] = queue[1:]
		}
		return []byte(item), nil
	}
}

func (s *scriptedProc) take(items *[]string, index *int) string {
	if len(*items) == 0 {
		return ""
	}
	if *index >= len(*items) {
		return (*items)[len(*items)-1]
	}
	item := (*items)[*index]
	*index++
	return item
}
