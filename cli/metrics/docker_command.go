/*
   Copyright 2022 Docker Compose CLI authors

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
	"strings"
	"time"
)

// DockerCLIEvent represents an invocation of `docker` from the the CLI.
type DockerCLIEvent struct {
	Command      string    `json:"command,omitempty"`
	Subcommand   string    `json:"subcommand,omitempty"`
	Usage        bool      `json:"usage,omitempty"`
	ExitCode     int32     `json:"exit_code"`
	StartTime    time.Time `json:"start_time"`
	DurationSecs float64   `json:"duration_secs,omitempty"`
}

// NewDockerCLIEvent inspects the command line string and returns a stripped down
// version suitable for reporting.
//
// The parser will only use known values for command/subcommand from a hardcoded
// built-in set for safety. It also does not attempt to perfectly accurately
// reflect how arg parsing works in a real program, instead favoring a fairly
// simple approach that's still reasonably robust.
//
// If the command does not map to a known Docker (or first-party plugin)
// command, `nil` will be returned. Similarly, if no subcommand for the
// built-in/plugin can be determined, it will be empty.
func NewDockerCLIEvent(cmd CmdResult) *DockerCLIEvent {
	if len(cmd.Args) == 0 {
		return nil
	}

	cmdPath := findCommand(append([]string{"docker"}, cmd.Args...))
	if cmdPath == nil {
		return nil
	}

	if len(cmdPath) < 2 {
		// ignore unknown commands; we can't infer anything from them safely
		// N.B. ONLY compose commands are supported by `cmdHierarchy` currently!
		return nil
	}

	// look for a subcommand
	var subcommand string
	if len(cmdPath) >= 3 {
		var subcommandParts []string
		for _, c := range cmdPath[2:] {
			subcommandParts = append(subcommandParts, c.name)
		}
		subcommand = strings.Join(subcommandParts, "-")
	}

	var usage bool
	for _, arg := range cmd.Args {
		// TODO(milas): also support `docker help build` syntax
		if arg == "help" {
			return nil
		}

		if arg == "--help" || arg == "-h" {
			usage = true
		}
	}

	event := &DockerCLIEvent{
		Command:      cmdPath[1].name,
		Subcommand:   subcommand,
		ExitCode:     int32(cmd.ExitCode),
		Usage:        usage,
		StartTime:    cmd.Start,
		DurationSecs: cmd.Duration.Seconds(),
	}

	return event
}

func findCommand(args []string) []*cmdNode {
	if len(args) == 0 {
		return nil
	}

	cmdPath := []*cmdNode{cmdHierarchy}
	if len(args) == 1 {
		return cmdPath
	}

	nodePath := []string{args[0]}
	for _, v := range args[1:] {
		v = strings.TrimSpace(v)
		if v == "" || strings.HasPrefix(v, "-") {
			continue
		}
		candidate := append(nodePath, v)
		if c := cmdHierarchy.find(candidate); c != nil {
			cmdPath = append(cmdPath, c)
			nodePath = candidate
		}
	}

	return cmdPath
}

type cmdNode struct {
	name     string
	plugin   bool
	children []*cmdNode
}

func (c *cmdNode) find(path []string) *cmdNode {
	if len(path) == 0 {
		return nil
	}

	if c.name != path[0] {
		return nil
	}

	if len(path) == 1 {
		return c
	}

	remainder := path[1:]
	for _, child := range c.children {
		if res := child.find(remainder); res != nil {
			return res
		}
	}

	return nil
}

var cmdHierarchy = &cmdNode{
	name: "docker",
	children: []*cmdNode{
		{
			name:   "compose",
			plugin: true,
			children: []*cmdNode{
				{
					name: "alpha",
					children: []*cmdNode{
						{name: "watch"},
						{name: "dryrun"},
					},
				},
				{name: "build"},
				{name: "config"},
				{name: "convert"},
				{name: "cp"},
				{name: "create"},
				{name: "down"},
				{name: "events"},
				{name: "exec"},
				{name: "images"},
				{name: "kill"},
				{name: "logs"},
				{name: "ls"},
				{name: "pause"},
				{name: "port"},
				{name: "ps"},
				{name: "pull"},
				{name: "push"},
				{name: "restart"},
				{name: "rm"},
				{name: "run"},
				{name: "start"},
				{name: "stop"},
				{name: "top"},
				{name: "unpause"},
				{name: "up"},
				{name: "version"},
			},
		},
	},
}
