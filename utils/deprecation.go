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

package utils

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
)

const deprecationMessage = "Docker Compose's integration for ECS and ACI will be retired in November 2023. Learn more: https://docs.docker.com/go/compose-ecs-eol/"

var warnOnce sync.Once

func ShowDeprecationWarning(w io.Writer) {
	warnOnce.Do(func() {
		if quiet, _ := strconv.ParseBool(os.Getenv("COMPOSE_CLOUD_EOL_SILENT")); quiet {
			return
		}
		_, _ = fmt.Fprintln(w, deprecationMessage)
	})
}
