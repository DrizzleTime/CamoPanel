package docker

import "context"

type Port struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}
type Container struct {
	ID      string   `json:"Id"`
	Names   []string `json:"Names"`
	Image   string   `json:"Image"`
	State   string   `json:"State"`
	Status  string   `json:"Status"`
	Created int64    `json:"Created"`
	Ports   []Port   `json:"Ports"`
}
type Image struct {
	ID       string   `json:"Id"`
	RepoTags []string `json:"RepoTags"`
	Size     int64    `json:"Size"`
	Created  int64    `json:"Created"`
}
type Network struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Driver string `json:"Driver"`
	Scope  string `json:"Scope"`
}
type Volume struct {
	Name       string `json:"Name"`
	Driver     string `json:"Driver"`
	Mountpoint string `json:"Mountpoint"`
}
type Engine struct {
	Name            string `json:"Name"`
	ServerVersion   string `json:"ServerVersion"`
	OperatingSystem string `json:"OperatingSystem"`
	NCPU            int    `json:"NCPU"`
	MemTotal        int64  `json:"MemTotal"`
}
type Snapshot struct {
	Mode        string      `json:"mode"`
	Engine      Engine      `json:"engine"`
	Containers  []Container `json:"containers"`
	Images      []Image     `json:"images"`
	Networks    []Network   `json:"networks"`
	Volumes     []Volume    `json:"volumes"`
	GeneratedAt string      `json:"generated_at"`
}
type PublishedPort struct {
	HostIP        string `json:"host_ip"`
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
}

type CreateRequest struct {
	Name             string          `json:"name" binding:"required,min=2,max=63"`
	Image            string          `json:"image" binding:"required,max=255"`
	HostPort         int             `json:"host_port" binding:"min=0,max=65535"`
	ContainerPort    int             `json:"container_port" binding:"min=0,max=65535"`
	Ports            []PublishedPort `json:"ports"`
	Env              []string        `json:"env"`
	Command          []string        `json:"command"`
	RestartPolicy    string          `json:"restart_policy"`
	Binds            []string        `json:"binds"`
	ConfirmHostMount bool            `json:"confirm_host_mount"`
	Pull             bool            `json:"pull"`
}

type ContainerDetail struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Image         string   `json:"image"`
	State         string   `json:"state"`
	Ports         []Port   `json:"ports"`
	Env           []string `json:"env"`
	Command       []string `json:"command"`
	RestartPolicy string   `json:"restart_policy"`
	Binds         []string `json:"binds"`
}

type Stats struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryBytes int64   `json:"memory_bytes"`
	MemoryLimit int64   `json:"memory_limit"`
}

type PullEvent struct {
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Image    string `json:"image"`
}
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

type RunContainerRequest struct {
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	Ports         []Port            `json:"ports"`
	Binds         []string          `json:"binds"`
	Env           []string          `json:"env"`
	RestartPolicy string            `json:"restart_policy"`
	Labels        map[string]string `json:"labels"`
}

type Service interface {
	Snapshot(context.Context) (Snapshot, error)
	Action(context.Context, string, string) error
	Create(context.Context, CreateRequest) (string, error)
	Logs(context.Context, string) (string, error)
	RunContainer(context.Context, RunContainerRequest) (string, error)
	FindContainerByName(context.Context, string) (*Container, error)
	Inspect(context.Context, string) (ContainerDetail, error)
	Stats(context.Context, string) (Stats, error)
	Pull(context.Context, string) ([]PullEvent, error)
	DeleteImage(context.Context, string) error
	PruneDangling(context.Context) (int, error)
	ForceRemove(context.Context, string) error
}
