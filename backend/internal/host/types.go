package host

import "time"

// Overview is one host sample plus the in-memory curve from the last hour.
type Overview struct {
	Mode          string    `json:"mode"`
	SampledAt     string    `json:"sampled_at"`
	UptimeSeconds float64   `json:"uptime_seconds"`
	CPU           CPU       `json:"cpu"`
	Memory        Memory    `json:"memory"`
	Swap          Memory    `json:"swap"`
	Load          Load      `json:"load"`
	Disks         []Disk    `json:"disks"`
	Networks      []Network `json:"networks"`
	History       []Point   `json:"history"`
}

type CPU struct {
	Percent float64 `json:"percent"`
	Cores   int     `json:"cores"`
}

type Memory struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	Percent   float64 `json:"percent"`
}

type Load struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type Disk struct {
	Mount   string  `json:"mount"`
	Device  string  `json:"device"`
	Total   uint64  `json:"total"`
	Used    uint64  `json:"used"`
	Percent float64 `json:"percent"`
}

type Network struct {
	Name     string  `json:"name"`
	RxBytes  uint64  `json:"rx_bytes"`
	TxBytes  uint64  `json:"tx_bytes"`
	RxPerSec float64 `json:"rx_per_sec"`
	TxPerSec float64 `json:"tx_per_sec"`
}

type Point struct {
	At         string  `json:"at"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
}

type Process struct {
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
	RSS        uint64  `json:"rss"`
}

// Sampler reads metrics from the local machine.
type Sampler interface {
	Overview(now time.Time) (Overview, error)
	Processes(now time.Time, limit int, sortBy string) ([]Process, error)
}
