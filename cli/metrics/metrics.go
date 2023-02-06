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
	"strings"

	"github.com/docker/compose/v2/pkg/utils"

	"github.com/docker/compose-cli/cli/metrics/metadata"
)

func (c *client) Track(context string, args []string, status string) {
	if isInvokedAsCliBackend() {
		return
	}
	command := GetCommand(args)
	if command != "" {
		c.SendUsage(Command{
			Command: command,
			Context: context,
			Source:  c.getMetadata(CLISource, args),
			Status:  status,
		})
	}
}

func (c *client) getMetadata(cliSource string, args []string) string {
	if len(args) == 0 {
		return cliSource
	}
	switch args[0] {
	case "build", "buildx":
		cliSource = metadata.BuildMetadata(cliSource, c.cliversion.f(), args[0], args[1:])
	}
	return cliSource
}

func isInvokedAsCliBackend() bool {
	executable := os.Args[0]
	return strings.HasSuffix(executable, "-backend")
}

func isCommand(word string) bool {
	return utils.StringContains(commands, word) || isManagementCommand(word)
}

func isManagementCommand(word string) bool {
	return utils.StringContains(managementCommands, word)
}

func isCommandFlag(word string) bool {
	return utils.StringContains(commandFlags, word)
}

// HasQuietFlag returns true if one of the arguments is `--quiet` or `-q`
func HasQuietFlag(args []string) bool {
	for _, a := range args {
		switch a {
		case "--quiet", "-q":
			return true
		}
	}
	return false
}

// GetCommand get the invoked command
func GetCommand(args []string) string {
	result := ""
	onlyFlags := false
	for _, arg := range args {
		if arg == "--help" {
			result = strings.TrimSpace(arg + " " + result)
			continue
		}
		if arg == "--" {
			break
		}
		if isCommandFlag(arg) || (!onlyFlags && isCommand(arg)) {
			result = strings.TrimSpace(result + " " + arg)
			if isCommand(arg) && !isManagementCommand(arg) {
				onlyFlags = true
			}
		}
	}
	return result
}
