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

package metadata

import (
	"io"
	"os"
	"testing"

	"github.com/docker/cli/cli/config"
	"gotest.tools/v3/assert"
)

func TestBuildxBuilder(t *testing.T) {
	tts := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "without builder",
			args:     []string{"buildx", "build", "-t", "foo:bar", "."},
			expected: "",
		},
		{
			name:     "with builder",
			args:     []string{"--builder", "foo", "buildx", "build", "."},
			expected: "foo",
		},
	}
	for _, tt := range tts {
		t.Run(tt.name, func(t *testing.T) {
			result := buildxBuilder(tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildxDriver(t *testing.T) {
	tts := []struct {
		name     string
		cfg      string
		args     []string
		expected string
	}{
		{
			name:     "no flag and default builder",
			cfg:      "./testdata/buildx-default",
			args:     []string{"buildx", "build", "-t", "foo:bar", "."},
			expected: "docker",
		},
		{
			name:     "no flag and current builder",
			cfg:      "./testdata/buildx-container",
			args:     []string{"buildx", "build", "-t", "foo:bar", "."},
			expected: "docker-container",
		},
		{
			name:     "builder flag",
			cfg:      "./testdata/buildx-default",
			args:     []string{"--builder", "graviton2", "buildx", "build", "."},
			expected: "docker-container",
		},
	}

	for _, tt := range tts {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("BUILDX_CONFIG", tt.cfg)
			result := buildxDriver(config.LoadDefaultConfigFile(io.Discard), tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBuildxDefault(t *testing.T) {
	tts := []struct {
		cliVersion string
		expected   bool
	}{
		{
			cliVersion: "",
			expected:   false,
		},
		{
			cliVersion: "20.10.15",
			expected:   false,
		},
		{
			cliVersion: "20.10.2-575-g22edabb584.m",
			expected:   false,
		},
		{
			cliVersion: "22.05.0",
			expected:   true,
		},
	}
	for _, tt := range tts {
		t.Run(tt.cliVersion, func(t *testing.T) {
			assert.Equal(t, tt.expected, isBuildxDefault(tt.cliVersion))
		})
	}
}
