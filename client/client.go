/*
	Copyright (c) 2019 Docker Inc.

	Permission is hereby granted, free of charge, to any person
	obtaining a copy of this software and associated documentation
	files (the "Software"), to deal in the Software without
	restriction, including without limitation the rights to use, copy,
	modify, merge, publish, distribute, sublicense, and/or sell copies
	of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be
	included in all copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
	EXPRESS OR IMPLIED,
	INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
	IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
	HOLDERS BE LIABLE FOR ANY CLAIM,
	DAMAGES OR OTHER LIABILITY,
	WHETHER IN AN ACTION OF CONTRACT,
	TORT OR OTHERWISE,
	ARISING FROM, OUT OF OR IN CONNECTION WITH
	THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package client

import (
	"context"
	"time"

	backendv1 "github.com/docker/api/backend/v1"
	containersv1 "github.com/docker/api/containers/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/protobuf/types/known/emptypb"
)

// New returns a GRPC client
func New(address string, timeout time.Duration) (*Client, error) {
	backoffConfig := backoff.DefaultConfig
	backoffConfig.MaxDelay = 3 * time.Second
	backoffConfig.BaseDelay = 10 * time.Millisecond
	connParams := grpc.ConnectParams{
		Backoff: backoffConfig,
	}
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithConnectParams(connParams),
		grpc.WithBlock(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:             conn,
		backendClient:    backendv1.NewBackendClient(conn),
		containersClient: containersv1.NewContainersClient(conn),
	}, nil
}

type Client struct {
	conn             *grpc.ClientConn
	backendClient    backendv1.BackendClient
	containersClient containersv1.ContainersClient
}

type BackendInformation struct {
	ID string
}

func (c *Client) BackendInformation(ctx context.Context) (BackendInformation, error) {
	info, err := c.backendClient.BackendInformation(ctx, &emptypb.Empty{})

	return BackendInformation{
		ID: info.Id,
	}, err
}

type Container struct {
	ID     string
	Status string
}

func (c *Client) List(ctx context.Context) ([]Container, error) {
	resp, err := c.containersClient.List(ctx, &containersv1.ListRequest{})
	if err != nil {
		// TODO: convert GRPC error
		return []Container{}, err
	}
	result := []Container{}
	for _, r := range resp.Containers {
		result = append(result, Container{
			ID:     r.Id,
			Status: r.Status,
		})
	}
	return result, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
