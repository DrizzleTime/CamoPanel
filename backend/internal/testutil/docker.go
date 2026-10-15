// Package testutil supplies isolated dependencies for backend tests.
package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"camopanel/backend/internal/docker"
)

// DockerCLI prevents tests from invoking the machine's Docker daemon.
func DockerCLI(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

type Docker struct {
	mu         sync.Mutex
	containers map[string]docker.ContainerDetail
	logs       map[string]string
	images     []docker.Image
	next       int
}

func NewDocker() *Docker {
	d := &Docker{containers: map[string]docker.ContainerDetail{}, logs: map[string]string{}, images: []docker.Image{{ID: "sha256:111111111111", RepoTags: []string{"nginx:alpine"}}, {ID: "sha256:444444444444", RepoTags: []string{"alpine:3.21"}}, {ID: "sha256:test-dangling"}}}
	for i, id := range []string{"a1b2c3d4e5f6", "b2c3d4e5f6a1", "c3d4e5f6a1b2"} {
		d.containers[id] = docker.ContainerDetail{ID: id, Name: fmt.Sprintf("test-%d", i), Image: "nginx:alpine", State: "running", Env: []string{}, Command: []string{}, Binds: []string{}, Ports: []docker.Port{}}
		d.logs[id] = "container initialized\n"
	}
	return d
}

func (d *Docker) Snapshot(context.Context) (docker.Snapshot, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	s := docker.Snapshot{Mode: "docker", Engine: docker.Engine{Name: "test-engine"}, Containers: []docker.Container{}, Images: append([]docker.Image{}, d.images...), Networks: []docker.Network{}, Volumes: []docker.Volume{{Name: "postgres-data"}}}
	for _, c := range d.containers {
		s.Containers = append(s.Containers, docker.Container{ID: c.ID, Names: []string{"/" + c.Name}, Image: c.Image, State: c.State, Ports: c.Ports})
	}
	return s, nil
}

func (d *Docker) Action(_ context.Context, id, action string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	c, ok := d.containers[id]
	if !ok {
		return &docker.Error{Status: 404, Message: "容器不存在"}
	}
	switch action {
	case "remove":
		if c.State == "running" {
			return &docker.Error{Status: 409, Message: "请先停止容器再删除"}
		}
		delete(d.containers, id)
		delete(d.logs, id)
		return nil
	case "stop":
		c.State = "exited"
	case "pause":
		c.State = "paused"
	default:
		c.State = "running"
	}
	d.containers[id] = c
	d.logs[id] += action + "\n"
	return nil
}

func (d *Docker) Create(_ context.Context, r docker.CreateRequest) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.next++
	id := fmt.Sprintf("%064x", d.next)
	for _, c := range d.containers {
		if c.Name == r.Name {
			return "", &docker.Error{Status: 409, Message: "容器名称已存在"}
		}
	}
	found := false
	for _, image := range d.images {
		for _, tag := range image.RepoTags {
			if tag == r.Image {
				found = true
			}
		}
	}
	if !found {
		return "", &docker.Error{Status: 404, Message: "请使用本地已有镜像"}
	}
	ports := []docker.Port{}
	for _, p := range r.Ports {
		ports = append(ports, docker.Port{IP: p.HostIP, PrivatePort: p.ContainerPort, PublicPort: p.HostPort, Type: p.Protocol})
	}
	if len(ports) == 0 && r.HostPort > 0 {
		ports = append(ports, docker.Port{IP: "127.0.0.1", PrivatePort: r.ContainerPort, PublicPort: r.HostPort, Type: "tcp"})
	}
	d.containers[id] = docker.ContainerDetail{ID: id, Name: r.Name, Image: r.Image, State: "created", Env: r.Env, Command: r.Command, RestartPolicy: r.RestartPolicy, Binds: r.Binds, Ports: ports}
	d.logs[id] = "container created\n"
	return id, nil
}

func (d *Docker) RunContainer(ctx context.Context, r docker.RunContainerRequest) (string, error) {
	_, _ = d.Pull(ctx, r.Image)
	id, err := d.Create(ctx, docker.CreateRequest{Name: r.Name, Image: r.Image, Env: r.Env, Binds: r.Binds, RestartPolicy: r.RestartPolicy})
	if err != nil {
		return "", err
	}
	return id, d.Action(ctx, id, "start")
}
func (d *Docker) FindContainerByName(_ context.Context, name string) (*docker.Container, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.containers {
		if c.Name == strings.TrimPrefix(name, "/") {
			return &docker.Container{ID: c.ID, Names: []string{"/" + c.Name}, Image: c.Image, State: c.State}, nil
		}
	}
	return nil, nil
}
func (d *Docker) Logs(_ context.Context, id string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if s, ok := d.logs[id]; ok {
		return s, nil
	}
	return "", &docker.Error{Status: 404, Message: "容器不存在"}
}
func (d *Docker) Inspect(_ context.Context, id string) (docker.ContainerDetail, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if c, ok := d.containers[id]; ok {
		return c, nil
	}
	return docker.ContainerDetail{}, &docker.Error{Status: 404, Message: "容器不存在"}
}
func (d *Docker) Stats(ctx context.Context, id string) (docker.Stats, error) {
	_, err := d.Inspect(ctx, id)
	return docker.Stats{CPUPercent: 1, MemoryBytes: 1024}, err
}
func (d *Docker) Pull(_ context.Context, image string) ([]docker.PullEvent, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.images = append(d.images, docker.Image{ID: image, RepoTags: []string{image}})
	return []docker.PullEvent{{Status: "Complete", Progress: 100, Image: image}}, nil
}
func (d *Docker) DeleteImage(_ context.Context, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, img := range d.images {
		if img.ID == id {
			d.images = append(d.images[:i], d.images[i+1:]...)
			return nil
		}
	}
	return &docker.Error{Status: 404, Message: "镜像不存在"}
}
func (d *Docker) PruneDangling(context.Context) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	kept := []docker.Image{}
	removed := 0
	for _, image := range d.images {
		if len(image.RepoTags) == 0 {
			removed++
			continue
		}
		kept = append(kept, image)
	}
	d.images = kept
	return removed, nil
}
func (d *Docker) ForceRemove(_ context.Context, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.containers, id)
	delete(d.logs, id)
	return nil
}

var _ docker.Service = (*Docker)(nil)
