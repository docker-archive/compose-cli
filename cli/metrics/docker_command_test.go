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
	"testing"

	"github.com/mattn/go-shellwords"
	"github.com/stretchr/testify/require"
)

func TestCmdCompose(t *testing.T) {
	tcs := []struct {
		name     string
		input    string
		expected *DockerCLIEvent
	}{
		{
			name:     "Docker - Root",
			input:    "docker",
			expected: nil,
		},
		{
			name:  "Docker - Build",
			input: "docker build .",
			// N.B. built-in commands are not reported currently, ONLY Compose
			expected: nil,
		},
		{
			name:     "Compose - Base",
			input:    "docker compose",
			expected: &DockerCLIEvent{Command: "compose"},
		},
		{
			name:  "Compose - Root Args",
			input: "docker --host 127.0.0.1 --debug=true compose ls",
			expected: &DockerCLIEvent{
				Command:    "compose",
				Subcommand: "ls",
			},
		},
		{
			name:  "Compose - Base Args",
			input: "docker compose -p myproject build myservice",
			expected: &DockerCLIEvent{
				Command:    "compose",
				Subcommand: "build",
			},
		},
		{
			name:  "Compose - Usage",
			input: "docker compose --file=mycompose.yaml up --help myservice",
			expected: &DockerCLIEvent{
				Command:    "compose",
				Subcommand: "up",
				Usage:      true,
			},
		},
		{
			name:  "Compose - Multilevel",
			input: "docker -D compose alpha --arg watch",
			expected: &DockerCLIEvent{
				Command:    "compose",
				Subcommand: "alpha-watch",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			args, err := shellwords.Parse(tc.input)
			require.NoError(t, err, "Invalid command: %s", tc.input)

			c := NewDockerCLIEvent(CmdResult{Args: args})
			require.Equal(t, tc.expected, c)
		})
	}
}
