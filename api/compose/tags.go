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

const (
	// ProjectTag allow to track resource related to a compose project
	ProjectTag = "com.docker.compose.project"
	// NetworkTag allow to track resource related to a compose network
	NetworkTag = "com.docker.compose.network"
	// ServiceTag allow to track resource related to a compose service
	ServiceTag = "com.docker.compose.service"
	// VolumeTag allow to track resource related to a compose volume
	VolumeTag = "com.docker.compose.volume"
	// EnvironmentFileLabel is set in containers with the option "--env-file" when set
	EnvironmentFileLabel = "com.docker.compose.project.environment_file"
)
