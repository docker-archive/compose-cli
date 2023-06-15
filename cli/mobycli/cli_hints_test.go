/*
   Copyright 2023 Docker Compose CLI authors

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

package mobycli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/compose-cli/api/config"

	"gotest.tools/v3/assert"
)

func TestCliHintsEnabled(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func()
		expected bool
	}{
		{
			"enabled by default",
			func() {},
			true,
		},
		{
			"enabled from environment variable",
			func() {
				t.Setenv(cliHintsEnvVarName, "true")
			},
			true,
		},
		{
			"disabled from environment variable",
			func() {
				t.Setenv(cliHintsEnvVarName, "false")
			},
			false,
		},
		{
			"unsupported value",
			func() {
				t.Setenv(cliHintsEnvVarName, "maybe")
			},
			true,
		},
		{
			"enabled in config file",
			func() {
				d := testConfigDir(t)
				writeSampleConfig(t, d, configEnabled)
			},
			true,
		},
		{
			"plugin defined in config file but no enabled entry",
			func() {
				d := testConfigDir(t)
				writeSampleConfig(t, d, configPartial)
			},
			true,
		},

		{
			"unsupported value",
			func() {
				d := testConfigDir(t)
				writeSampleConfig(t, d, configOnce)
			},
			true,
		},
		{
			"disabled in config file",
			func() {
				d := testConfigDir(t)
				writeSampleConfig(t, d, configDisabled)
			},
			false,
		},
		{
			"enabled in config file but disabled by env var",
			func() {
				d := testConfigDir(t)
				writeSampleConfig(t, d, configEnabled)
				t.Setenv(cliHintsEnvVarName, "false")
			},
			false,
		},
		{
			"disabled in config file but enabled by env var",
			func() {
				d := testConfigDir(t)
				writeSampleConfig(t, d, configDisabled)
				t.Setenv(cliHintsEnvVarName, "true")
			},
			true,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			assert.Equal(t, CliHintsEnabled(), tc.expected)
		})
	}
}

func testConfigDir(t *testing.T) string {
	dir := config.Dir()
	d, _ := os.MkdirTemp("", "")
	config.WithDir(d)
	t.Cleanup(func() {
		_ = os.RemoveAll(d)
		config.WithDir(dir)
	})
	return d
}

func writeSampleConfig(t *testing.T, d string, conf []byte) {
	err := os.WriteFile(filepath.Join(d, config.ConfigFileName), conf, 0644)
	assert.NilError(t, err)
}

var configEnabled = []byte(`{
	"plugins": {
		"-x-cli-hints": {
			"enabled": "true"
		}
	}
}`)

var configDisabled = []byte(`{
	"plugins": {
		"-x-cli-hints": {
			"enabled": "false"
		}
	}
}`)

var configPartial = []byte(`{
	"plugins": {
		"-x-cli-hints": {
		}
	}
}`)

var configOnce = []byte(`{
	"plugins": {
		"-x-cli-hints": {
			"enabled": "maybe"
		}
	}
}`)
