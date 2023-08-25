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
	"strings"
	"time"

	"github.com/docker/cli/cli/config"
	"github.com/docker/compose/v2/pkg/utils"
	"github.com/docker/docker/registry"

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
	if _, ok := os.LookupEnv("BUILDKIT_PROGRESS"); ok {
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

	_, _ = b.Fprintln(out, "\nWhat's Next?")
	if !hubLoggedIn() {
		_, _ = fmt.Fprintln(out, "  1. Sign in to your Docker account → "+color.CyanString("docker login"))
		_, _ = fmt.Fprintln(out, "  2. View a summary of image vulnerabilities and recommendations → "+color.CyanString("docker scout quickview"+image))
	} else {
		_, _ = fmt.Fprintln(out, "  View a summary of image vulnerabilities and recommendations → "+color.CyanString("docker scout quickview"+image))
	}
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

// hubLoggedIn checks whether the user has credentials configured
// for Docker Hub. If it fails to get a result within 100ms, it
// short-circuits and returns `true`.
// This can be an expensive operation, so use it mindfully.
func hubLoggedIn() bool {
	result := make(chan bool)
	go func() {
		creds, err := config.LoadDefaultConfigFile(nil).GetAllCredentials()
		if err != nil {
			// preserve original behaviour if we fail to fetch creds
			result <- true
		}
		_, ok := creds[registry.IndexServer]
		result <- ok
	}()
	select {
	case loggedIn := <-result:
		return loggedIn
	case <-time.After(100 * time.Millisecond):
		// preserve original behaviour if we time out
		return true
	}
}
