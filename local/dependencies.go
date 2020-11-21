// +build local

/*
   Copyright 2020 Docker Compose CLI authors

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

package local

import (
	"context"

	"github.com/compose-spec/compose-go/types"
	"golang.org/x/sync/errgroup"
)

func inDependencyOrder(ctx context.Context, project *types.Project, fn func(context.Context, types.ServiceConfig) error) error {
	graph := buildDependencyGraph(project.Services)
	return graph.walk(ctx, fn).Wait()
}

type dependencyGraph map[string]node

type node struct {
	service      types.ServiceConfig
	dependencies []string
	dependent    []string
}

func (graph dependencyGraph) filter(predicate func(node) bool) []node {
	var filtered []node
	for _, n := range graph {
		if predicate(n) {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

func withoutDepencies(n node) bool {
	return len(n.dependencies) == 0
}

func (graph dependencyGraph) walk(ctx context.Context, fn func(context.Context, types.ServiceConfig) error) *errgroup.Group {
	eg, ctx := errgroup.WithContext(ctx)
	resultsCh := make(chan node)
	schedule := func(ctx context.Context, n node) {
		eg.Go(func() error {
			err := fn(ctx, n.service)
			if err != nil {
				return err
			}
			resultsCh <- n
			return nil
		})
	}

	// start producer goroutine
	go func() {
		visited := []string{}
		for len(visited) < len(graph) {
			done := <-resultsCh
			visited = append(visited, done.service.Name)
			for _, n := range done.dependent {
				dependent := graph[n]
				if containsAll(visited, dependent.dependencies) {
					schedule(ctx, dependent)
				}
			}
		}
	}()

	for _, n := range graph.filter(withoutDepencies) {
		schedule(ctx, n)
	}
	return eg
}

func buildDependencyGraph(services types.Services) dependencyGraph {
	graph := dependencyGraph{}
	for _, s := range services {
		graph[s.Name] = node{
			service: s,
		}
	}

	for _, s := range services {
		node := graph[s.Name]
		for _, name := range s.GetDependencies() {
			dependency := graph[name]
			node.dependencies = append(node.dependencies, name)
			dependency.dependent = append(dependency.dependent, s.Name)
			graph[name] = dependency
		}
		graph[s.Name] = node
	}
	return graph
}
