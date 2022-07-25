//go:build !windows && !darwin
// +build !windows,!darwin

/*
   Copyright 2020, 2022 Docker Compose CLI authors

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
	"net"
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/homedir"
)

var (
	socket = "/var/run/docker-cli.sock"
)

func init() {
	// Attempt to retrieve the Docker CLI socket for the current user.
	if home := homedir.Get(); home != "" {
		tmp := filepath.Join(home, ".docker/desktop/docker-cli.sock")
		if _, err := os.Stat(tmp); err == nil {
			socket = tmp
		} // else: fall back to the global CLI socket path (used by DD in WSL)
	} // else: fall back to the global CLI socket path (used by DD in WSL)
	overrideSocket() // no-op, unless built for e2e testing
}

func conn() (net.Conn, error) {
	return net.Dial("unix", socket)
}
