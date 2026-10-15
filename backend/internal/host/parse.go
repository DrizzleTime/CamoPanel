package host

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func round1(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return math.Round(v*10) / 10
}

type cpuTimes struct {
	idle  uint64
	total uint64
	cores int
}

func parseCPUStat(data string) (cpuTimes, error) {
	var times cpuTimes
	found := false
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || !strings.HasPrefix(fields[0], "cpu") {
			continue
		}
		if fields[0] != "cpu" {
			times.cores++
			continue
		}
		nums := make([]uint64, 0, len(fields)-1)
		for _, field := range fields[1:] {
			n, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				return cpuTimes{}, fmt.Errorf("解析 CPU 统计失败")
			}
			nums = append(nums, n)
		}
		var total uint64
		for _, n := range nums {
			total += n
		}
		idle := nums[3]
		if len(nums) > 4 {
			idle += nums[4]
		}
		times.idle = idle
		times.total = total
		found = true
	}
	if !found {
		return cpuTimes{}, fmt.Errorf("解析 CPU 统计失败")
	}
	if times.cores == 0 {
		times.cores = 1
	}
	return times, nil
}

func cpuPercent(prev, cur cpuTimes) float64 {
	if cur.total < prev.total || cur.idle < prev.idle {
		return 0
	}
	dt := cur.total - prev.total
	if dt == 0 {
		return 0
	}
	busy := float64(dt-(cur.idle-prev.idle)) / float64(dt) * 100
	if busy > 100 {
		busy = 100
	}
	return round1(busy)
}

func parseMeminfo(data string) (mem Memory, swap Memory, err error) {
	values := map[string]uint64{}
	for _, line := range strings.Split(data, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) == 0 {
			continue
		}
		n, convErr := strconv.ParseUint(fields[0], 10, 64)
		if convErr != nil {
			continue
		}
		if len(fields) > 1 && fields[1] == "kB" {
			n *= 1024
		}
		values[parts[0]] = n
	}
	total := values["MemTotal"]
	if total == 0 {
		return Memory{}, Memory{}, fmt.Errorf("解析内存信息失败")
	}
	available := values["MemAvailable"]
	if available == 0 {
		available = values["MemFree"] + values["Buffers"] + values["Cached"]
	}
	if available > total {
		available = total
	}
	mem = memoryPair(total, available)
	swapTotal := values["SwapTotal"]
	swapFree := values["SwapFree"]
	if swapFree > swapTotal {
		swapFree = swapTotal
	}
	swap = memoryPair(swapTotal, swapFree)
	return mem, swap, nil
}

func memoryPair(total, available uint64) Memory {
	if available > total {
		available = total
	}
	used := total - available
	percent := 0.0
	if total > 0 {
		percent = round1(float64(used) / float64(total) * 100)
	}
	return Memory{Total: total, Used: used, Available: available, Percent: percent}
}

func parseLoadavg(data string) (Load, error) {
	fields := strings.Fields(data)
	if len(fields) < 3 {
		return Load{}, fmt.Errorf("解析负载失败")
	}
	var load Load
	var err error
	if load.Load1, err = strconv.ParseFloat(fields[0], 64); err != nil {
		return Load{}, fmt.Errorf("解析负载失败")
	}
	if load.Load5, err = strconv.ParseFloat(fields[1], 64); err != nil {
		return Load{}, fmt.Errorf("解析负载失败")
	}
	if load.Load15, err = strconv.ParseFloat(fields[2], 64); err != nil {
		return Load{}, fmt.Errorf("解析负载失败")
	}
	return load, nil
}

func parseUptime(data string) (float64, error) {
	fields := strings.Fields(data)
	if len(fields) < 1 {
		return 0, fmt.Errorf("解析运行时间失败")
	}
	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || uptime < 0 {
		return 0, fmt.Errorf("解析运行时间失败")
	}
	return uptime, nil
}

type netCounters struct {
	rx uint64
	tx uint64
}

func parseNetDev(data string) (map[string]netCounters, error) {
	lines := strings.Split(data, "\n")
	if len(lines) < 3 {
		return nil, fmt.Errorf("解析网络统计失败")
	}
	out := map[string]netCounters{}
	for _, line := range lines[2:] {
		name, payload, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(payload)
		if len(fields) < 9 {
			continue
		}
		rx, rxErr := strconv.ParseUint(fields[0], 10, 64)
		tx, txErr := strconv.ParseUint(fields[8], 10, 64)
		if rxErr != nil || txErr != nil {
			return nil, fmt.Errorf("解析网络统计失败")
		}
		out[strings.TrimSpace(name)] = netCounters{rx: rx, tx: tx}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("解析网络统计失败")
	}
	return out, nil
}

type mountEntry struct {
	device string
	mount  string
	fsType string
}

func parseMounts(data string) []mountEntry {
	skip := map[string]bool{
		"proc": true, "sysfs": true, "devtmpfs": true, "tmpfs": true, "devpts": true,
		"cgroup": true, "cgroup2": true, "pstore": true, "bpf": true, "tracefs": true,
		"debugfs": true, "securityfs": true, "mqueue": true, "hugetlbfs": true,
		"overlay": true, "squashfs": true, "autofs": true, "binfmt_misc": true,
		"configfs": true, "fusectl": true, "ramfs": true, "nsfs": true, "rpc_pipefs": true,
	}
	best := map[string]mountEntry{}
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || skip[fields[2]] {
			continue
		}
		device := unescapeMount(fields[0])
		mount := unescapeMount(fields[1])
		if !strings.HasPrefix(mount, "/") {
			continue
		}
		current, ok := best[device]
		if !ok || len(mount) < len(current.mount) {
			best[device] = mountEntry{device: device, mount: mount, fsType: fields[2]}
		}
	}
	list := make([]mountEntry, 0, len(best))
	for _, entry := range best {
		list = append(list, entry)
	}
	return list
}

func unescapeMount(value string) string {
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return replacer.Replace(value)
}

func parseProcStat(data string) (name string, ticks uint64, err error) {
	start := strings.IndexByte(data, '(')
	end := strings.LastIndexByte(data, ')')
	if start < 0 || end <= start {
		return "", 0, fmt.Errorf("解析进程失败")
	}
	rest := strings.Fields(data[end+1:])
	// Fields after comm: state is index 0. utime is field 14 (index 11), stime is field 15 (index 12).
	if len(rest) < 13 {
		return "", 0, fmt.Errorf("解析进程失败")
	}
	utime, uErr := strconv.ParseUint(rest[11], 10, 64)
	stime, sErr := strconv.ParseUint(rest[12], 10, 64)
	if uErr != nil || sErr != nil {
		return "", 0, fmt.Errorf("解析进程失败")
	}
	return data[start+1 : end], utime + stime, nil
}

func parseStatus(data string) (name string, rss uint64) {
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(line, "Name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		}
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, convErr := strconv.ParseUint(fields[1], 10, 64)
				if convErr == nil {
					rss = kb * 1024
				}
			}
		}
	}
	return name, rss
}

func stamped(now time.Time) string {
	return now.UTC().Format(time.RFC3339)
}
