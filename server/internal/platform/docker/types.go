package docker

import (
	"context"
	"time"
)

const (
	StatusNotFound = "not_found"
	StatusRunning  = "running"
	StatusStopped  = "stopped"
	StatusDegraded = "degraded"
)

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

type Container struct {
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

type Image struct {
	ID         string    `json:"id"`
	RepoTags   []string  `json:"repo_tags"`
	Containers int64     `json:"containers"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
}

type ImagePruneResult struct {
	ImagesDeleted  int    `json:"images_deleted"`
	SpaceReclaimed uint64 `json:"space_reclaimed"`
}

type Network struct {
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

type SystemInfo struct {
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

type Reader interface {
	ListContainers(ctx context.Context) ([]Container, error)
	ListImages(ctx context.Context) ([]Image, error)
	ListNetworks(ctx context.Context) ([]Network, error)
	GetSystemInfo(ctx context.Context) (SystemInfo, error)
	ContainerLogs(ctx context.Context, containerID string, tail int) (string, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	RestartContainer(ctx context.Context, containerID string) error
	DeleteContainer(ctx context.Context, containerID string) error
	RemoveImage(ctx context.Context, imageID string) error
	PruneUnusedImages(ctx context.Context) (ImagePruneResult, error)
}

type Runtime interface {
	EnsureNetwork(ctx context.Context, name, driver string) error
	Deploy(ctx context.Context, projectName, composePath string) error
	Start(ctx context.Context, projectName, composePath string) error
	Stop(ctx context.Context, projectName, composePath string) error
	Restart(ctx context.Context, projectName, composePath string) error
	Redeploy(ctx context.Context, projectName, composePath string) error
	Delete(ctx context.Context, projectName, composePath string) error
	InspectProject(ctx context.Context, projectName string) (ProjectRuntime, error)
	ProjectLogs(ctx context.Context, projectName string, tail int) (string, error)
}

type ContainerOperator interface {
	InspectContainer(ctx context.Context, containerName string) (ContainerStatus, error)
	ExecInContainer(ctx context.Context, containerName string, args ...string) (string, error)
}
