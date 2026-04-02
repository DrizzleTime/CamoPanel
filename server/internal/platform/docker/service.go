package docker

import (
	"bytes"
	"context"
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

type Service struct {
	client        *client.Client
	clientErr     error
	dockerBinPath string
}

func NewService() *Service {
	dockerBin, _ := exec.LookPath("docker")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return &Service{clientErr: err, dockerBinPath: dockerBin}
	}
	return &Service{
		client:        cli,
		dockerBinPath: dockerBin,
	}
}

func (d *Service) EnsureNetwork(ctx context.Context, name, driver string) error {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return err
	}

	items, err := cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", "^"+name+"$")),
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	for _, item := range items {
		if item.Name == name {
			return nil
		}
	}

	if _, err := cli.NetworkCreate(ctx, name, network.CreateOptions{Driver: driver}); err != nil && !errdefs.IsConflict(err) {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

func (d *Service) Deploy(ctx context.Context, projectName, composePath string) error {
	return d.runCompose(ctx, projectName, composePath, "up", "-d")
}

func (d *Service) Start(ctx context.Context, projectName, composePath string) error {
	return d.runCompose(ctx, projectName, composePath, "start")
}

func (d *Service) Stop(ctx context.Context, projectName, composePath string) error {
	return d.runCompose(ctx, projectName, composePath, "stop")
}

func (d *Service) Restart(ctx context.Context, projectName, composePath string) error {
	return d.runCompose(ctx, projectName, composePath, "restart")
}

func (d *Service) Redeploy(ctx context.Context, projectName, composePath string) error {
	if err := d.runCompose(ctx, projectName, composePath, "pull"); err != nil {
		return err
	}
	return d.runCompose(ctx, projectName, composePath, "up", "-d")
}

func (d *Service) Delete(ctx context.Context, projectName, composePath string) error {
	return d.runCompose(ctx, projectName, composePath, "down", "--remove-orphans")
}

func (d *Service) InspectProject(ctx context.Context, projectName string) (ProjectRuntime, error) {
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
		return ProjectRuntime{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	runtime := ProjectRuntime{
		Status:     StatusNotFound,
		Containers: make([]ProjectContainer, 0, len(containerList)),
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
	runtime.Status = deriveProjectStatus(runtime.Containers)
	return runtime, nil
}

func deriveProjectStatus(containers []ProjectContainer) string {
	if len(containers) == 0 {
		return StatusNotFound
	}

	running := 0
	exited := 0
	for _, item := range containers {
		switch item.State {
		case "running":
			running++
		case "exited", "dead":
			exited++
		}
	}

	switch {
	case running == len(containers):
		return StatusRunning
	case exited == len(containers):
		return StatusStopped
	default:
		return StatusDegraded
	}
}

func (d *Service) ListContainers(ctx context.Context) ([]Container, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return nil, err
	}

	containerList, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	items := make([]Container, 0, len(containerList))
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

		items = append(items, Container{
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

func (d *Service) ListImages(ctx context.Context) ([]Image, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return nil, err
	}

	imageList, err := cli.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	items := make([]Image, 0, len(imageList))
	for _, item := range imageList {
		repoTags := append([]string(nil), item.RepoTags...)
		sort.Strings(repoTags)
		items = append(items, Image{
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

func (d *Service) ListNetworks(ctx context.Context) ([]Network, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return nil, err
	}

	networkList, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	items := make([]Network, 0, len(networkList))
	for _, item := range networkList {
		containerCount := 0
		if item.Containers != nil {
			containerCount = len(item.Containers)
		}
		items = append(items, Network{
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

func (d *Service) GetSystemInfo(ctx context.Context) (SystemInfo, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return SystemInfo{}, err
	}

	info, err := cli.Info(ctx)
	if err != nil {
		return SystemInfo{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
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

	return SystemInfo{
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

func (d *Service) ProjectLogs(ctx context.Context, projectName string, tail int) (string, error) {
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

func (d *Service) ContainerLogs(ctx context.Context, containerID string, tail int) (string, error) {
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
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer reader.Close()

	var output bytes.Buffer
	_, _ = stdcopy.StdCopy(&output, &output, reader)
	return strings.TrimSpace(output.String()), nil
}

func (d *Service) StartContainer(ctx context.Context, containerID string) error {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return err
	}

	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		if errdefs.IsNotFound(err) {
			return ErrContainerNotFound
		}
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

func (d *Service) StopContainer(ctx context.Context, containerID string) error {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return err
	}

	if err := cli.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		if errdefs.IsNotFound(err) {
			return ErrContainerNotFound
		}
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

func (d *Service) RestartContainer(ctx context.Context, containerID string) error {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return err
	}

	if err := cli.ContainerRestart(ctx, containerID, container.StopOptions{}); err != nil {
		if errdefs.IsNotFound(err) {
			return ErrContainerNotFound
		}
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

func (d *Service) DeleteContainer(ctx context.Context, containerID string) error {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return err
	}

	if err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		if errdefs.IsNotFound(err) {
			return ErrContainerNotFound
		}
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

func (d *Service) RemoveImage(ctx context.Context, imageID string) error {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return err
	}

	if _, err := cli.ImageRemove(ctx, imageID, image.RemoveOptions{Force: false, PruneChildren: true}); err != nil {
		if errdefs.IsNotFound(err) {
			return ErrImageNotFound
		}
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

func (d *Service) PruneUnusedImages(ctx context.Context) (ImagePruneResult, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return ImagePruneResult{}, err
	}

	report, err := cli.ImagesPrune(ctx, filters.NewArgs(filters.Arg("dangling", "false")))
	if err != nil {
		return ImagePruneResult{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return ImagePruneResult{
		ImagesDeleted:  len(report.ImagesDeleted),
		SpaceReclaimed: report.SpaceReclaimed,
	}, nil
}

func (d *Service) InspectContainer(ctx context.Context, containerName string) (ContainerStatus, error) {
	cli, err := d.clientOrErr(ctx)
	if err != nil {
		return ContainerStatus{}, err
	}

	info, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return ContainerStatus{Name: containerName, Exists: false}, nil
		}
		return ContainerStatus{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
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

func (d *Service) ExecInContainer(ctx context.Context, containerName string, args ...string) (string, error) {
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
			return "", fmt.Errorf("%w: 未找到容器 %s", ErrUnavailable, containerName)
		}
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	attachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer attachResp.Close()

	var output bytes.Buffer
	_, _ = stdcopy.StdCopy(&output, &output, attachResp.Reader)

	inspectResp, err := cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
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

func (d *Service) clientOrErr(ctx context.Context) (*client.Client, error) {
	if d.clientErr != nil || d.client == nil {
		return nil, ErrUnavailable
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err := d.client.Ping(pingCtx); err != nil {
		return nil, ErrUnavailable
	}
	return d.client, nil
}

func (d *Service) runCompose(ctx context.Context, projectName, composePath string, args ...string) error {
	if d.dockerBinPath == "" {
		return ErrUnavailable
	}

	cmdArgs := append([]string{"compose", "-p", projectName, "-f", composePath}, args...)
	cmd := exec.CommandContext(ctx, d.dockerBinPath, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		return fmt.Errorf("%w: %s", ErrUnavailable, trimmed)
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
