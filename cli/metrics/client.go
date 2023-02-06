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

package metrics

import (
	"os"
	"time"
)

// EnvVarDebugMetricsPath is an optional environment variable used to debug
// metrics by triggering all events to also be written as JSON lines to the
// specified file path.
const EnvVarDebugMetricsPath = "DOCKER_METRICS_DEBUG_LOG"

// Timeout is the maximum amount of time we'll wait for metrics sending to be
// acknowledged before giving up.
const Timeout = 50 * time.Millisecond

// CmdResult provides details about process execution.
type CmdResult struct {
	// ContextType is `moby` for Docker or the name of a cloud provider.
	ContextType string
	// Args minus the process name (argv[0] aka `docker`).
	Args []string
	// Status based on exit code as a descriptive value.
	//
	// Deprecated: used for usage, events rely exclusively on exit code.
	Status string
	// ExitCode is 0 on success; otherwise, failure.
	ExitCode int
	// Start time of the process (UTC).
	Start time.Time
	// Duration of process execution.
	Duration time.Duration
}

type client struct {
	cliversion *cliversion
	reporter   Reporter
}

type cliversion struct {
	f func() string
}

// CommandUsage reports a CLI invocation for aggregation.
type CommandUsage struct {
	Command string `json:"command"`
	Context string `json:"context"`
	Source  string `json:"source"`
	Status  string `json:"status"`
}

// CLISource is sent for cli metrics
var CLISource = "cli"

func init() {
	if v, ok := os.LookupEnv("DOCKER_METRICS_SOURCE"); ok {
		CLISource = v
	}
}

// Client sends metrics to Docker Desktop
type Client interface {
	// WithCliVersionFunc sets the docker cli version func
	// that returns the docker cli version (com.docker.cli)
	WithCliVersionFunc(f func() string)
	// SendUsage sends the command to Docker Desktop.
	//
	// Note that metric collection is best-effort, so any errors are ignored.
	SendUsage(CommandUsage)
	// Track creates an event for a command execution and reports it.
	Track(CmdResult)
}

// NewClient returns a new metrics client that will send metrics using the
// provided Reporter instance.
func NewClient(reporter Reporter) Client {
	return &client{
		cliversion: &cliversion{},
		reporter:   reporter,
	}
}

// NewDefaultClient returns a new metrics client that will send metrics using
// the default Reporter configuration, which reports via HTTP, and, optionally,
// to a local file for debugging. (No format guarantees are made!)
func NewDefaultClient() Client {
	httpClient := newHTTPClient()

	var reporter Reporter = NewHTTPReporter(httpClient)
	if metricsLogPath := os.Getenv(EnvVarDebugMetricsPath); metricsLogPath != "" {
		if f, err := os.OpenFile(metricsLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err != nil {
			panic(err)
		} else {
			reporter = NewMuxReporter(
				NewWriterReporter(f),
				reporter,
			)
		}
	}
	return NewClient(reporter)
}

func (c *client) WithCliVersionFunc(f func() string) {
	c.cliversion.f = f
}

func (c *client) SendUsage(command CommandUsage) {
	result := make(chan bool, 1)
	go func() {
		c.reporter.Heartbeat(command)
		result <- true
	}()

	// wait for the post finished, or timeout in case anything freezes.
	// Posting metrics without Desktop listening returns in less than a ms, and a handful of ms (often <2ms) when Desktop is listening
	select {
	case <-result:
	case <-time.After(Timeout):
	}
}
