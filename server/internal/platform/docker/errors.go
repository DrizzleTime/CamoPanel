package docker

import "errors"

var (
	ErrUnavailable       = errors.New("docker is unavailable")
	ErrContainerNotFound = errors.New("container not found")
	ErrImageNotFound     = errors.New("image not found")
)
