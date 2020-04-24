package containers

import (
	"context"

	containersv1 "github.com/docker/api/containers/v1"
)

type Container struct {
	ID          string
	Status      string
	Image       string
	CpuTime     uint64
	MemoryUsage uint64
	MemoryLimit uint64
	PidsCurrent uint64
	PidsLimit   uint64
	Labels      []string
}

type ContainerService interface {
	List(ctx context.Context) ([]Container, error)
}

func NewContainerApi(client containersv1.ContainersClient) ContainerService {
	return &containerApi{
		client: client,
	}
}

type containerApi struct {
	client containersv1.ContainersClient
}

func fromGrpc(c *containersv1.Container) Container {
	return Container{
		ID:          c.Id,
		Status:      c.Status,
		Image:       c.Image,
		CpuTime:     c.CpuTime,
		MemoryUsage: c.MemoryUsage,
		MemoryLimit: c.MemoryLimit,
		PidsCurrent: c.PidsCurrent,
		PidsLimit:   c.PidsLimit,
		Labels:      c.Labels,
	}
}

func (c *containerApi) List(ctx context.Context) ([]Container, error) {
	resp, err := c.client.List(ctx, &containersv1.ListRequest{})
	if err != nil {
		// TODO: convert GRPC error
		return []Container{}, err
	}

	result := []Container{}
	for _, c := range resp.Containers {
		result = append(result, fromGrpc(c))
	}

	return result, nil
}
