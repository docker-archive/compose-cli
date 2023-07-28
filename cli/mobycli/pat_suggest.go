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
	"strings"

	"github.com/docker/docker/registry"
	"github.com/hashicorp/go-uuid"
)

var (
	patPrefixes = []string{"dckrp_", "dckr_pat_"}
)

func isUsingDefaultRegistry(cmdArgs []string) bool {
	for i := 1; i < len(cmdArgs); i++ {
		if strings.HasPrefix(cmdArgs[i], "-") {
			i++
			continue
		}
		return cmdArgs[i] == registry.IndexServer
	}
	return true
}

func isUsingPassword(pass string) bool {
	if pass == "" { // ignore if no password (or SSO)
		return false
	}
	if _, err := uuid.ParseUUID(pass); err == nil {
		return false
	}
	for _, patPrefix := range patPrefixes {
		if strings.HasPrefix(pass, patPrefix) {
			return false
		}
	}
	return true
}
