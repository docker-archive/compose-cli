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
	"strings"

	"github.com/docker/compose/v2/pkg/utils"

	"github.com/fatih/color"
)

func displayScoutQuickViewSuggestMsgOnPull(args []string) {
	image := pulledImageFromArgs(args)
	displayScoutQuickViewSuggestMsg(image)
}

func displayScoutQuickViewSuggestMsgOnBuild(args []string) {
	// only display the hint in the main case, build command and not buildx build, no output flag, no progress flag, no push flag
	if utils.StringContains(args, "--output") || utils.StringContains(args, "-o") ||
		utils.StringContains(args, "--progress") ||
		utils.StringContains(args, "--push") {
		return
	}
	displayScoutQuickViewSuggestMsg("")
}

func displayScoutQuickViewSuggestMsg(image string) {
	if !CliHintsEnabled() {
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

func pulledImageFromArgs(args []string) string {
	var image string
	var pull bool
	for _, a := range args {
		if a == "pull" {
			pull = true
			continue
		}
		if pull && !strings.HasPrefix(a, "-") {
			image = a
			break
		}
	}
	return image
}
