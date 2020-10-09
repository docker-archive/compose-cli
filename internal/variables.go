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

package internal

// UNDEFINED is the placeholder to variables that have to be specified at build time
const UNDEFINED = `Undefined! To be injected in the build. Please check "vars.mk" and "builder.Makefile"`

const (
	// UserAgentName is the default user agent used by the cli
	UserAgentName = "docker-cli"
	// ECSUserAgentName is the ECS specific user agent used by the cli
	ECSUserAgentName = "Docker CLI"
)

// The variables below are injected on build time
var (
	// Version is the version of the CLI injected in compilation time
	Version = "dev"
	// ACIDNSSidecarImage is the image used by the side car container in ACI
	ACIDNSSidecarImage = UNDEFINED
)
