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
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
)

const (
	scoutHintEnvVarName = "DOCKER_SCOUT_HINTS"
)

func isDockerScoutHintsEnabled() bool {
	enabled, err := strconv.ParseBool(os.Getenv(scoutHintEnvVarName))
	return err != nil || enabled
}

func displayScoutQuickViewSuggestMsg(image string) {
	if !isDockerScoutHintsEnabled() {
		return
	}
	if len(image) > 0 {
		image = " " + image
	}
	out := os.Stderr
	b := color.New(color.Bold)
	_, _ = fmt.Fprintln(out)
	_, _ = b.Fprintln(out, "What's Next?")
	_, _ = fmt.Fprintf(out, "  View summary of image vulnerabilities and recommendations → %s", color.CyanString("docker scout quickview%s", image))
	_, _ = fmt.Fprintln(out)
}
