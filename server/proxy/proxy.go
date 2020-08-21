/*
   Copyright 2020 Docker, Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package proxy

import (
	"context"
	"sync"

	"github.com/docker/api/client"
	"github.com/docker/api/config"
	containersv1 "github.com/docker/api/protos/containers/v1"
	contextsv1 "github.com/docker/api/protos/contexts/v1"
	streamsv1 "github.com/docker/api/protos/streams/v1"
	"github.com/docker/api/server/proxy/streams"
)

type clientKey struct{}

// WithClient adds the client to the context
func WithClient(ctx context.Context, c *client.Client) (context.Context, error) {
	return context.WithValue(ctx, clientKey{}, c), nil
}

// Client returns the client from the context
func Client(ctx context.Context) *client.Client {
	c, _ := ctx.Value(clientKey{}).(*client.Client)
	return c
}

// Proxy implements the gRPC server and forwards the actions
// to the right backend
type Proxy interface {
	containersv1.ContainersServiceServer
	streamsv1.StreamingServiceServer
	ContextsProxy() contextsv1.ContextsServiceServer
}

type proxy struct {
	configDir     string
	mu            sync.Mutex
	streams       map[string]*streams.Stream
	contextsProxy *contextsProxy
}

// New creates a new proxy server
func New(ctx context.Context) Proxy {
	configDir := config.Dir(ctx)
	return &proxy{
		configDir: configDir,
		streams:   map[string]*streams.Stream{},
		contextsProxy: &contextsProxy{
			configDir: configDir,
		},
	}
}

func (p *proxy) ContextsProxy() contextsv1.ContextsServiceServer {
	return p.contextsProxy
}
