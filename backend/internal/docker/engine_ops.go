package docker

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

func (e *EngineClient) Inspect(ctx context.Context, id string) (ContainerDetail, error) {
	prefix, err := e.prefix(ctx)
	if err != nil {
		return ContainerDetail{}, err
	}
	b, err := e.request(ctx, "GET", prefix+"/containers/"+url.PathEscape(id)+"/json", nil)
	if err != nil {
		return ContainerDetail{}, err
	}
	var raw struct {
		ID     string `json:"Id"`
		Name   string `json:"Name"`
		Image  string `json:"Image"`
		Config struct {
			Image string   `json:"Image"`
			Env   []string `json:"Env"`
			Cmd   []string `json:"Cmd"`
		} `json:"Config"`
		HostConfig struct {
			Binds         []string `json:"Binds"`
			RestartPolicy struct {
				Name string `json:"Name"`
			} `json:"RestartPolicy"`
		} `json:"HostConfig"`
		State struct {
			Status string `json:"Status"`
		} `json:"State"`
		NetworkSettings struct {
			Ports map[string][]struct {
				HostIP   string `json:"HostIp"`
				HostPort string `json:"HostPort"`
			} `json:"Ports"`
		} `json:"NetworkSettings"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return ContainerDetail{}, err
	}
	ports := []Port{}
	for key, bindings := range raw.NetworkSettings.Ports {
		private, proto, _ := strings.Cut(key, "/")
		n, _ := strconv.Atoi(private)
		if len(bindings) == 0 {
			ports = append(ports, Port{PrivatePort: n, Type: proto})
			continue
		}
		for _, binding := range bindings {
			pub, _ := strconv.Atoi(binding.HostPort)
			ports = append(ports, Port{IP: binding.HostIP, PrivatePort: n, PublicPort: pub, Type: proto})
		}
	}
	image := raw.Config.Image
	if image == "" {
		image = raw.Image
	}
	env, cmd, binds := raw.Config.Env, raw.Config.Cmd, raw.HostConfig.Binds
	if env == nil {
		env = []string{}
	}
	if cmd == nil {
		cmd = []string{}
	}
	if binds == nil {
		binds = []string{}
	}
	return ContainerDetail{ID: raw.ID, Name: strings.TrimPrefix(raw.Name, "/"), Image: image, State: raw.State.Status, Ports: ports, Env: env, Command: cmd, RestartPolicy: raw.HostConfig.RestartPolicy.Name, Binds: binds}, nil
}

func (e *EngineClient) Stats(ctx context.Context, id string) (Stats, error) {
	prefix, err := e.prefix(ctx)
	if err != nil {
		return Stats{}, err
	}
	b, err := e.request(ctx, "GET", prefix+"/containers/"+url.PathEscape(id)+"/stats?stream=false", nil)
	if err != nil {
		return Stats{}, err
	}
	var raw struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs  int    `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
		} `json:"memory_stats"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return Stats{}, err
	}
	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(raw.CPUStats.SystemUsage - raw.PreCPUStats.SystemUsage)
	cpus := raw.CPUStats.OnlineCPUs
	if cpus < 1 {
		cpus = 1
	}
	percent := 0.0
	if sysDelta > 0 && cpuDelta >= 0 {
		percent = (cpuDelta / sysDelta) * float64(cpus) * 100
	}
	return Stats{CPUPercent: percent, MemoryBytes: int64(raw.MemoryStats.Usage), MemoryLimit: int64(raw.MemoryStats.Limit)}, nil
}

func (e *EngineClient) Pull(ctx context.Context, image string) ([]PullEvent, error) {
	prefix, err := e.prefix(ctx)
	if err != nil {
		return nil, err
	}
	name, tag, _ := strings.Cut(image, ":")
	if tag == "" {
		tag = "latest"
	}
	b, err := e.request(ctx, "POST", prefix+"/images/create?fromImage="+url.QueryEscape(name)+"&tag="+url.QueryEscape(tag), nil)
	if err != nil {
		return nil, err
	}
	events := []PullEvent{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Status   string `json:"status"`
			Progress string `json:"progress"`
		}
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		progress := 50
		if strings.Contains(strings.ToLower(row.Status), "complete") || strings.Contains(strings.ToLower(row.Status), "downloaded") || strings.Contains(strings.ToLower(row.Status), "up to date") {
			progress = 100
		}
		events = append(events, PullEvent{Status: row.Status, Progress: progress, Image: image})
	}
	if len(events) == 0 {
		events = []PullEvent{{Status: "Complete", Progress: 100, Image: image}}
	}
	return events, nil
}

func (e *EngineClient) DeleteImage(ctx context.Context, id string) error {
	prefix, err := e.prefix(ctx)
	if err != nil {
		return err
	}
	_, err = e.request(ctx, "DELETE", prefix+"/images/"+url.PathEscape(id), nil)
	return err
}

func (e *EngineClient) PruneDangling(ctx context.Context) (int, error) {
	prefix, err := e.prefix(ctx)
	if err != nil {
		return 0, err
	}
	b, err := e.request(ctx, "POST", prefix+"/images/prune?filters="+url.QueryEscape(`{"dangling":{"true":true}}`), nil)
	if err != nil {
		return 0, err
	}
	var raw struct {
		ImagesDeleted []struct {
			Deleted string `json:"Deleted"`
		} `json:"ImagesDeleted"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return 0, err
	}
	return len(raw.ImagesDeleted), nil
}

func (e *EngineClient) ForceRemove(ctx context.Context, id string) error {
	prefix, err := e.prefix(ctx)
	if err != nil {
		return err
	}
	_, err = e.request(ctx, "DELETE", prefix+"/containers/"+url.PathEscape(id)+"?force=true", nil)
	return err
}
