/*
   Copyright 2021 Docker Compose CLI authors

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

package compose

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/filters"

	"github.com/docker/compose-cli/api/compose"
)

const (
	containerNumberLabel = "com.docker.compose.container-number"
	oneoffLabel          = "com.docker.compose.oneoff"
	slugLabel            = "com.docker.compose.slug"
	projectLabel         = compose.ProjectTag
	volumeLabel          = compose.VolumeTag
	workingDirLabel      = "com.docker.compose.project.working_dir"
	configFilesLabel     = "com.docker.compose.project.config_files"
	serviceLabel         = compose.ServiceTag
	versionLabel         = "com.docker.compose.version"
	configHashLabel      = "com.docker.compose.config-hash"
	networkLabel         = compose.NetworkTag

	// ComposeVersion Compose version
	ComposeVersion = "1.0-alpha"
)

func projectFilter(projectName string) filters.KeyValuePair {
	return filters.Arg("label", fmt.Sprintf("%s=%s", projectLabel, projectName))
}

func oneOffFilter(oneOff bool) filters.KeyValuePair {
	return filters.Arg("label", fmt.Sprintf("%s=%s", oneoffLabel, strings.Title(strconv.FormatBool(oneOff))))
}

func serviceFilter(serviceName string) filters.KeyValuePair {
	return filters.Arg("label", fmt.Sprintf("%s=%s", serviceLabel, serviceName))
}

func hasProjectLabelFilter() filters.KeyValuePair {
	return filters.Arg("label", projectLabel)
}
