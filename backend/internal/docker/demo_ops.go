package docker

import (
	"context"
	"strings"
	"time"
)

func (d *Demo) Inspect(_ context.Context, id string) (ContainerDetail, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.data.Containers {
		if c.ID != id && !strings.HasPrefix(c.ID, id) {
			continue
		}
		spec := d.specs[c.ID]
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		env, command, binds := spec.Env, spec.Command, spec.Binds
		if env == nil {
			env = []string{}
		}
		if command == nil {
			command = []string{}
		}
		if binds == nil {
			binds = []string{}
		}
		return ContainerDetail{ID: c.ID, Name: name, Image: c.Image, State: c.State, Ports: append([]Port{}, c.Ports...), Env: env, Command: command, RestartPolicy: spec.Restart, Binds: binds}, nil
	}
	return ContainerDetail{}, &Error{404, "容器不存在"}
}

func (d *Demo) Stats(_ context.Context, id string) (Stats, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.data.Containers {
		if c.ID != id {
			continue
		}
		n := 0
		for _, ch := range c.ID {
			n += int(ch)
		}
		return Stats{CPUPercent: float64(n%40) + 1.5, MemoryBytes: int64((n%180)+32) << 20, MemoryLimit: 512 << 20}, nil
	}
	return Stats{}, &Error{404, "容器不存在"}
}

func (d *Demo) Pull(_ context.Context, image string) ([]PullEvent, error) {
	image = strings.TrimSpace(image)
	if image == "" {
		return nil, &Error{400, "镜像名不能为空"}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	found := false
	for _, item := range d.data.Images {
		for _, tag := range item.RepoTags {
			if tag == image {
				found = true
			}
		}
	}
	if !found {
		d.data.Images = append(d.data.Images, Image{ID: "sha256:pull" + strings.ReplaceAll(image, ":", ""), RepoTags: []string{image}, Size: 20 << 20, Created: time.Now().Unix()})
	}
	return []PullEvent{{Status: "Pulling", Progress: 30, Image: image}, {Status: "Download", Progress: 80, Image: image}, {Status: "Complete", Progress: 100, Image: image}}, nil
}

func (d *Demo) DeleteImage(_ context.Context, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, image := range d.data.Images {
		if image.ID == id || strings.HasPrefix(image.ID, id) {
			d.data.Images = append(d.data.Images[:i], d.data.Images[i+1:]...)
			return nil
		}
	}
	return &Error{404, "镜像不存在"}
}

func dangling(image Image) bool {
	if len(image.RepoTags) == 0 {
		return true
	}
	for _, tag := range image.RepoTags {
		if tag == "" || strings.Contains(tag, "<none>") {
			return true
		}
	}
	return false
}

func (d *Demo) PruneDangling(_ context.Context) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	kept := make([]Image, 0, len(d.data.Images))
	removed := 0
	for _, image := range d.data.Images {
		if dangling(image) {
			removed++
			continue
		}
		kept = append(kept, image)
	}
	d.data.Images = kept
	return removed, nil
}

func (d *Demo) ForceRemove(_ context.Context, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, c := range d.data.Containers {
		if c.ID != id {
			continue
		}
		d.data.Containers = append(d.data.Containers[:i], d.data.Containers[i+1:]...)
		delete(d.logs, id)
		delete(d.specs, id)
		return nil
	}
	return &Error{404, "容器不存在"}
}
