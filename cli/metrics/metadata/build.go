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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/spf13/pflag"
)

// getBuildMetadata returns build metadata for this command
func getBuildMetadata(cliSource string, command string, args []string) string {
	var cli, builder string
	dockercfg := config.LoadDefaultConfigFile(io.Discard)
	if alias, ok := dockercfg.Aliases["builder"]; ok {
		command = alias
	}
	if command == "build" {
		cli = "docker"
		builder = "buildkit"
		if enabled, _ := isBuildKitEnabled(); !enabled {
			builder = "legacy"
		}
	} else if command == "buildx" {
		cli = "buildx"
		builder = buildxDriver(dockercfg, args)
	}
	return fmt.Sprintf("%s-%s;%s", cliSource, cli, builder)
}

// isBuildKitEnabled returns whether buildkit is enabled either through a
// daemon setting or otherwise the client-side DOCKER_BUILDKIT environment
// variable
func isBuildKitEnabled() (bool, error) {
	if buildkitEnv := os.Getenv("DOCKER_BUILDKIT"); len(buildkitEnv) > 0 {
		return strconv.ParseBool(buildkitEnv)
	}
	apiClient, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return false, err
	}
	defer apiClient.Close() //nolint:errcheck
	ping, err := apiClient.Ping(context.Background())
	if err != nil {
		return false, err
	}
	return ping.BuilderVersion == types.BuilderBuildKit, nil
}

// buildxConfigDir will look for correct configuration store path;
// if `$BUILDX_CONFIG` is set - use it, otherwise use parent directory
// of Docker config file (i.e. `${DOCKER_CONFIG}/buildx`)
func buildxConfigDir(dockercfg *configfile.ConfigFile) string {
	if buildxConfig := os.Getenv("BUILDX_CONFIG"); buildxConfig != "" {
		return buildxConfig
	}
	return filepath.Join(filepath.Dir(dockercfg.Filename), "buildx")
}

// buildxDriver returns the build driver being used for the build command
func buildxDriver(dockercfg *configfile.ConfigFile, buildArgs []string) string {
	driver := "error"
	configDir := buildxConfigDir(dockercfg)
	if _, err := os.Stat(configDir); err != nil {
		return driver
	}
	builder := buildxBuilder(buildArgs)
	if len(builder) == 0 {
		// if builder not defined in command, seek current in buildx store
		// `${DOCKER_CONFIG}/buildx/current`
		fileCurrent := path.Join(configDir, "current")
		if _, err := os.Stat(fileCurrent); err != nil {
			return driver
		}
		// content looks like
		// {
		//   "Key": "unix:///var/run/docker.sock",
		//   "Name": "builder",
		//   "Global": false
		// }
		rawCurrent, err := ioutil.ReadFile(fileCurrent)
		if err != nil {
			return driver
		}
		// unmarshal and returns `Name`
		var obj map[string]interface{}
		if err = json.Unmarshal(rawCurrent, &obj); err != nil {
			return driver
		}
		if n, ok := obj["Name"]; ok {
			builder = n.(string)
			// `Name` will be empty if `default` builder is used
			// {
			//   "Key": "unix:///var/run/docker.sock",
			//   "Name": "",
			//   "Global": false
			// }
			if len(builder) == 0 {
				builder = "default"
			}
		} else {
			return driver
		}
	}

	// if default builder return docker
	if builder == "default" {
		return "docker"
	}

	// read builder info and retrieve the current driver
	// `${DOCKER_CONFIG}/buildx/instances/<builder>`
	fileBuilder := path.Join(configDir, "instances", builder)
	if _, err := os.Stat(fileBuilder); err != nil {
		return driver
	}
	// content looks like
	// {
	//   "Name": "builder",
	//   "Driver": "docker-container",
	//   "Nodes": [
	//     {
	//       "Name": "builder0",
	//       "Endpoint": "unix:///var/run/docker.sock",
	//       "Platforms": null,
	//       "Flags": null,
	//       "ConfigFile": "",
	//       "DriverOpts": null
	//     }
	//   ],
	//   "Dynamic": false
	// }
	rawBuilder, err := ioutil.ReadFile(fileBuilder)
	if err != nil {
		return driver
	}
	// unmarshal and returns `Driver`
	var obj map[string]interface{}
	if err = json.Unmarshal(rawBuilder, &obj); err != nil {
		return driver
	}
	if d, ok := obj["Driver"]; ok {
		driver = d.(string)
	}
	return driver
}

// buildxBuilder returns the builder being used in the build command
func buildxBuilder(buildArgs []string) string {
	var builder string
	flags := pflag.NewFlagSet("buildx", pflag.ContinueOnError)
	flags.Usage = func() {}
	flags.StringVar(&builder, "builder", "", "")
	_ = flags.Parse(buildArgs)
	if len(builder) == 0 {
		builder = os.Getenv("BUILDX_BUILDER")
	}
	return builder
}
