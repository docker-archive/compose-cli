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

package mobycli

import (
	"os"
	"strconv"

	"github.com/docker/compose-cli/api/config"
)

const (
	cliHintsEnvVarName       = "DOCKER_CLI_HINTS"
	cliHintsDefaultBehaviour = true
)

func CliHintsEnabled() bool {
	if envValue, ok := os.LookupEnv(cliHintsEnvVarName); ok {
		if enabled, err := strconv.ParseBool(envValue); err == nil {
			return enabled
		}
	}

	conf, err := config.LoadFile(config.Dir())
	if err != nil {
		// can't read the config file, use the default behaviour
		return cliHintsDefaultBehaviour
	}
	if conf.CliHints != nil {
		return *conf.CliHints
	}

	return cliHintsDefaultBehaviour
}
