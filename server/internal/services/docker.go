package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"
)

var ErrDockerUnavailable = errors.New("docker is unavailable")
var ErrContainerNotFound = errors.New("container not found")

type ProjectContainer struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Image  string   `json:"image"`
	State  string   `json:"state"`
	Status string   `json:"status"`
	Ports  []string `json:"ports"`
}

type ProjectRuntime struct {
	Status     string             `json:"status"`
	Containers []ProjectContainer `json:"containers"`
}

type ContainerMount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type ContainerStatus struct {
	Exists  bool             `json:"exists"`
	Running bool             `json:"running"`
	Name    string           `json:"name"`
	Image   string           `json:"image"`
	Status  string           `json:"status"`
	Mounts  []ContainerMount `json:"mounts"`
}

type ContainerOperator interface {
	InspectContainer(ctx context.Context, containerName string) (ContainerStatus, error)
	ExecInContainer(ctx context.Context, containerName string, args ...string) (string, error)
}

type DockerContainer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Image     string    `json:"image"`
	State     string    `json:"state"`
	Status    string    `json:"status"`
	Project   string    `json:"project"`
	Ports     []string  `json:"ports"`
	Networks  []string  `json:"networks"`
	CreatedAt time.Time `json:"created_at"`
}

type DockerImage struct {
	ID         string    `json:"id"`
	RepoTags   []string  `json:"repo_tags"`
	Containers int64     `json:"containers"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
}

type DockerNetwork struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Driver         string    `json:"driver"`
	Scope          string    `json:"scope"`
	Internal       bool      `json:"internal"`
	Attachable     bool      `json:"attachable"`
	Ingress        bool      `json:"ingress"`
	ContainerCount int       `json:"container_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type DockerSystemInfo struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	ServerVersion     string   `json:"server_version"`
	OperatingSystem   string   `json:"operating_system"`
	KernelVersion     string   `json:"kernel_version"`
	Architecture      string   `json:"architecture"`
	NCPU              int      `json:"ncpu"`
	MemTotal          int64    `json:"mem_total"`
	DockerRootDir     string   `json:"docker_root_dir"`
	Driver            string   `json:"driver"`
	LoggingDriver     string   `json:"logging_driver"`
	CgroupDriver      string   `json:"cgroup_driver"`
	CgroupVersion     string   `json:"cgroup_version"`
	DefaultRuntime    string   `json:"default_runtime"`
	Runtimes          []string `json:"runtimes"`
	NetworkPlugins    []string `json:"network_plugins"`
	VolumePlugins     []string `json:"volume_plugins"`
	Containers        int      `json:"containers"`
	ContainersRunning int      `json:"containers_running"`
	ContainersPaused  int      `json:"containers_paused"`
	ContainersStopped int      `json:"containers_stopped"`
	Images            int      `json:"images"`
	Warnings          []string `json:"warnings"`
}

type DockerReader interface {
	ListContainers(ctx context.Context) ([]DockerContainer, error)
	ListImages(ctx context.Context) ([]DockerImage, error)
	ListNetworks(ctx context.Context) ([]DockerNetwork, error)
	GetSystemInfo(ctx context.Context) (DockerSystemInfo, error)
	ContainerLogs(ctx context.Context, containerID string, tail int) (string, error)
}

type Executor interface {
	Deploy(ctx context.Context, projectName, composePath string) error
	Start(ctx context.Context, projectName, composePath string) error
	Stop(ctx context.Context, projectName, composePath string) error
	Restart(ctx context.Context, projectName, composePath string) error
	Redeploy(ctx context.Context, projectName, composePath string) error
	Delete(ctx context.Context, projectName, composePath string) error
	InspectProject(ctx context.Context, projectName string) (ProjectRuntime, error)
	ProjectLogs(ctx context.Context, projectName string, tail int) (string, error)
}

type DockerService struct {
	client        *client.Client
	clientErr     error
	dockerBinPath string
}

func NewDockerService() *DockerService {
	dockerBin, _ := exec.LookPath("docker")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return &DockerService{clientErr: err, dockerBinPath: dockerBin}
	}
	return &DockerService{
		client:        cli,
		dockerBinPath: dockerBin,
	}
}

func (d *DockerService) Deploy(ctx context.Context, projectName, composePath string) error {
	return d.runCompose(ctx, projectName, composePath, "up", "-d")
}

func (d *DockerService) Start(ctx context.Context, projectName, composePath string) error {
	return d.runCompose(ctx, projectName, composePath, "start")
}

func (d *DockerService) Stop(ctx context.Context, projectName, composePath string) error {
	return d.runCompose(ctx, projectName, composePath, "stop")
}

func (d *DockerService) Restart(ctx context.Context, projectName, composePath string) error {
	return d.runCompose(ctx, projectName, composePath, "restart")
}

func (d *DockerService) Redeploy(ctx context.Context, projectName, composePath string) error {
	if err := d.runCompose(ctx, projectName, composePath, "pull"); err != nil {
		return err
	}
	return d.runCompose(ctx, projectName, composePath, "up", "-d")
}

func (d *DockerService) Delete(ctx context.Context, projectName, composePath string) error {
	return d.runCompose(ctx, projectName, composePath, "down", "--remove-orphans")
}

func (d *DockerService) InspectProject(ctx context.Context, projectName string) (ProjectRuntime, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return ProjectRuntime{}, err
	}

	containerList, err := cli.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", "com.docker.compose.project="+projectName),
		),
	})
	if err != nil {
		return ProjectRuntime{}, fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
	}

	runtime := ProjectRuntime{
		Status:     "not_found",
		Containers: []ProjectContainer{},
	}

	for _, item := range containerList {
		runtime.Containers = append(runtime.Containers, ProjectContainer{
			ID:     item.ID,
			Name:   containerName(item),
			Image:  item.Image,
			State:  item.State,
			Status: item.Status,
			Ports:  formatContainerPorts(item.Ports),
		})
	}

	sort.Slice(runtime.Containers, func(i, j int) bool {
		return runtime.Containers[i].Name < runtime.Containers[j].Name
	})

	if len(runtime.Containers) == 0 {
		return runtime, nil
	}

	running := 0
	exited := 0
	for _, item := range runtime.Containers {
		switch item.State {
		case "running":
			running++
		case "exited", "dead":
			exited++
		}
	}

	switch {
	case running == len(runtime.Containers):
		runtime.Status = "running"
	case exited == len(runtime.Containers):
		runtime.Status = "stopped"
	default:
		runtime.Status = "degraded"
	}

	return runtime, nil
}

func (d *DockerService) ListContainers(ctx context.Context) ([]DockerContainer, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return nil, err
	}

	containerList, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
	}

	items := make([]DockerContainer, 0, len(containerList))
	for _, item := range containerList {
		networks := []string{}
		if item.NetworkSettings != nil {
			for name := range item.NetworkSettings.Networks {
				networks = append(networks, name)
			}
		}
		if len(networks) == 0 && item.HostConfig.NetworkMode != "" {
			networks = append(networks, item.HostConfig.NetworkMode)
		}
		sort.Strings(networks)

		items = append(items, DockerContainer{
			ID:        item.ID,
			Name:      containerName(item),
			Image:     item.Image,
			State:     item.State,
			Status:    item.Status,
			Project:   strings.TrimSpace(item.Labels["com.docker.compose.project"]),
			Ports:     formatContainerPorts(item.Ports),
			Networks:  networks,
			CreatedAt: time.Unix(item.Created, 0).UTC(),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (d *DockerService) ListImages(ctx context.Context) ([]DockerImage, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return nil, err
	}

	imageList, err := cli.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
	}

	items := make([]DockerImage, 0, len(imageList))
	for _, item := range imageList {
		repoTags := append([]string(nil), item.RepoTags...)
		sort.Strings(repoTags)

		items = append(items, DockerImage{
			ID:         item.ID,
			RepoTags:   repoTags,
			Containers: item.Containers,
			Size:       item.Size,
			CreatedAt:  time.Unix(item.Created, 0).UTC(),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	return items, nil
}

func (d *DockerService) ListNetworks(ctx context.Context) ([]DockerNetwork, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return nil, err
	}

	networkList, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
	}

	items := make([]DockerNetwork, 0, len(networkList))
	for _, item := range networkList {
		containerCount := 0
		if item.Containers != nil {
			containerCount = len(item.Containers)
		}

		items = append(items, DockerNetwork{
			ID:             item.ID,
			Name:           item.Name,
			Driver:         item.Driver,
			Scope:          item.Scope,
			Internal:       item.Internal,
			Attachable:     item.Attachable,
			Ingress:        item.Ingress,
			ContainerCount: containerCount,
			CreatedAt:      item.Created.UTC(),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (d *DockerService) GetSystemInfo(ctx context.Context) (DockerSystemInfo, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return DockerSystemInfo{}, err
	}

	info, err := cli.Info(ctx)
	if err != nil {
		return DockerSystemInfo{}, fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
	}

	runtimes := make([]string, 0, len(info.Runtimes))
	for name := range info.Runtimes {
		runtimes = append(runtimes, name)
	}
	sort.Strings(runtimes)

	networkPlugins := append([]string(nil), info.Plugins.Network...)
	volumePlugins := append([]string(nil), info.Plugins.Volume...)
	warnings := append([]string(nil), info.Warnings...)
	sort.Strings(networkPlugins)
	sort.Strings(volumePlugins)

	return DockerSystemInfo{
		ID:                info.ID,
		Name:              info.Name,
		ServerVersion:     info.ServerVersion,
		OperatingSystem:   info.OperatingSystem,
		KernelVersion:     info.KernelVersion,
		Architecture:      info.Architecture,
		NCPU:              info.NCPU,
		MemTotal:          info.MemTotal,
		DockerRootDir:     info.DockerRootDir,
		Driver:            info.Driver,
		LoggingDriver:     info.LoggingDriver,
		CgroupDriver:      info.CgroupDriver,
		CgroupVersion:     info.CgroupVersion,
		DefaultRuntime:    info.DefaultRuntime,
		Runtimes:          runtimes,
		NetworkPlugins:    networkPlugins,
		VolumePlugins:     volumePlugins,
		Containers:        info.Containers,
		ContainersRunning: info.ContainersRunning,
		ContainersPaused:  info.ContainersPaused,
		ContainersStopped: info.ContainersStopped,
		Images:            info.Images,
		Warnings:          warnings,
	}, nil
}

func (d *DockerService) ProjectLogs(ctx context.Context, projectName string, tail int) (string, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return "", err
	}

	runtime, err := d.InspectProject(ctx, projectName)
	if err != nil {
		return "", err
	}
	if len(runtime.Containers) == 0 {
		return "", nil
	}

	var sections []string
	for _, item := range runtime.Containers {
		reader, err := cli.ContainerLogs(ctx, item.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       fmt.Sprintf("%d", tail),
			Timestamps: false,
		})
		if err != nil {
			sections = append(sections, fmt.Sprintf("[%s]\nfailed to read logs: %v", item.Name, err))
			continue
		}

		var stdout bytes.Buffer
		_, _ = stdcopy.StdCopy(&stdout, &stdout, reader)
		_ = reader.Close()
		sections = append(sections, fmt.Sprintf("[%s]\n%s", item.Name, strings.TrimSpace(stdout.String())))
	}

	return strings.TrimSpace(strings.Join(sections, "\n\n")), nil
}

func (d *DockerService) ContainerLogs(ctx context.Context, containerID string, tail int) (string, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return "", err
	}

	reader, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", max(tail, 1)),
		Timestamps: false,
	})
	if err != nil {
		if errdefs.IsNotFound(err) {
			return "", ErrContainerNotFound
		}
		return "", fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
	}
	defer reader.Close()

	var output bytes.Buffer
	_, _ = stdcopy.StdCopy(&output, &output, reader)
	return strings.TrimSpace(output.String()), nil
}

func (d *DockerService) InspectContainer(ctx context.Context, containerName string) (ContainerStatus, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return ContainerStatus{}, err
	}

	info, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return ContainerStatus{Name: containerName, Exists: false}, nil
		}
		return ContainerStatus{}, fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
	}

	status := ContainerStatus{
		Exists: true,
		Name:   strings.TrimPrefix(info.Name, "/"),
	}
	if info.Config != nil {
		status.Image = info.Config.Image
	}
	if info.State != nil {
		status.Running = info.State.Running
		status.Status = info.State.Status
	}

	for _, mount := range info.Mounts {
		status.Mounts = append(status.Mounts, ContainerMount{
			Source:      mount.Source,
			Destination: mount.Destination,
		})
	}

	sort.Slice(status.Mounts, func(i, j int) bool {
		return status.Mounts[i].Destination < status.Mounts[j].Destination
	})

	return status, nil
}

func (d *DockerService) ExecInContainer(ctx context.Context, containerName string, args ...string) (string, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return "", err
	}

	execResp, err := cli.ContainerExecCreate(ctx, containerName, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          args,
	})
	if err != nil {
		if errdefs.IsNotFound(err) {
			return "", fmt.Errorf("%w: 未找到容器 %s", ErrDockerUnavailable, containerName)
		}
		return "", fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
	}

	attachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
	}
	defer attachResp.Close()

	var output bytes.Buffer
	_, _ = stdcopy.StdCopy(&output, &output, attachResp.Reader)

	inspectResp, err := cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
	}

	trimmed := strings.TrimSpace(output.String())
	if inspectResp.ExitCode != 0 {
		if trimmed == "" {
			return "", fmt.Errorf("容器内命令执行失败，退出码 %d", inspectResp.ExitCode)
		}
		return "", fmt.Errorf("容器内命令执行失败: %s", trimmed)
	}

	return trimmed, nil
}

func (d *DockerService) clientOrErr(ctx context.Context) (*client.Client, error) {
	if d.clientErr != nil || d.client == nil {
		return nil, ErrDockerUnavailable
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err := d.client.Ping(pingCtx); err != nil {
		return nil, ErrDockerUnavailable
	}
	return d.client, nil
}

func (d *DockerService) runCompose(ctx context.Context, projectName, composePath string, args ...string) error {
	if d.dockerBinPath == "" {
		return ErrDockerUnavailable
	}

	cmdArgs := append([]string{"compose", "-p", projectName, "-f", composePath}, args...)
	cmd := exec.CommandContext(ctx, d.dockerBinPath, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
		}
		return fmt.Errorf("%w: %s", ErrDockerUnavailable, trimmed)
	}

	return nil
}

func formatContainerPorts(ports []container.Port) []string {
	items := make([]string, 0, len(ports))
	for _, port := range ports {
		if port.PublicPort > 0 {
			items = append(items, fmt.Sprintf("%d:%d/%s", port.PublicPort, port.PrivatePort, port.Type))
			continue
		}
		items = append(items, fmt.Sprintf("%d/%s", port.PrivatePort, port.Type))
	}

	sort.Strings(items)
	return items
}

func containerName(item container.Summary) string {
	if len(item.Names) == 0 {
		return item.ID
	}
	return strings.TrimPrefix(item.Names[0], "/")
}
