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
	"fmt"
	"os"

	"github.com/docker/compose-cli/api/config"
)

const (
	cliHintsEnvVarName       = "DOCKER_CLI_HINTS"
	cliHintsDefaultBehaviour = true

	cliHintsPluginName  = "-x-cli-hints"
	cliHintsEnabledName = "enabled"
	cliHintsEnabled     = "true"
	cliHintsDisabled    = "false"
)

func CliHintsEnabled() bool {
	if envValue, ok := os.LookupEnv(cliHintsEnvVarName); ok {
		if enabled, err := parseCliHintFlag(envValue); err == nil {
			return enabled
		}
	}

	conf, err := config.LoadFile(config.Dir())
	if err != nil {
		// can't read the config file, use the default behaviour
		return cliHintsDefaultBehaviour
	}
	if cliHintsPluginConfig, ok := conf.Plugins[cliHintsPluginName]; ok {
		if cliHintsValue, ok := cliHintsPluginConfig[cliHintsEnabledName]; ok {
			if cliHints, err := parseCliHintFlag(cliHintsValue); err == nil {
				return cliHints
			}
		}
	}

	return cliHintsDefaultBehaviour
}

func parseCliHintFlag(value string) (bool, error) {
	switch value {
	case cliHintsEnabled:
		return true, nil
	case cliHintsDisabled:
		return false, nil
	default:
		return cliHintsDefaultBehaviour, fmt.Errorf("could not parse CLI hints enabled flag")
	}
}
