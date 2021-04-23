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

package e2e

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	. "github.com/docker/compose-cli/utils/e2e"
)

func TestLocalComposeVolume(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	const projectName = "compose-e2e-volume"

	t.Run("up with build and no image name, volume", func(t *testing.T) {
		// ensure local test run does not reuse previously build image
		c.RunDockerOrExitError("rmi", "compose-e2e-volume_nginx")
		c.RunDockerOrExitError("volume", "rm", projectName+"_staticVol")
		c.RunDockerOrExitError("volume", "rm", "myvolume")
		c.RunDockerCmd("compose", "--project-directory", "fixtures/volume-test", "--project-name", projectName, "up", "-d")
	})

	t.Run("access bind mount data", func(t *testing.T) {
		output := HTTPGetWithRetry(t, "http://localhost:8090", http.StatusOK, 2*time.Second, 20*time.Second)
		assert.Assert(t, strings.Contains(output, "Hello from Nginx container"))
	})

	t.Run("check container volume specs", func(t *testing.T) {
		res := c.RunDockerCmd("inspect", "compose-e2e-volume_nginx2_1", "--format", "{{ json .Mounts }}")
		output := res.Stdout()
		// nolint
		assert.Assert(t, strings.Contains(output, `"Destination":"/usr/src/app/node_modules","Driver":"local","Mode":"z","RW":true,"Propagation":""`), output)
		assert.Assert(t, strings.Contains(output, `"Destination":"/myconfig","Mode":"ro","RW":false,"Propagation":"rprivate"`), output)
	})

	t.Run("check config content", func(t *testing.T) {
		output := c.RunDockerCmd("exec", "compose-e2e-volume_nginx2_1", "cat", "/myconfig").Stdout()
		assert.Assert(t, strings.Contains(output, `Hello from Nginx container`), output)
	})

	t.Run("check secrets content", func(t *testing.T) {
		output := c.RunDockerCmd("exec", "compose-e2e-volume_nginx2_1", "cat", "/run/secrets/mysecret").Stdout()
		assert.Assert(t, strings.Contains(output, `Hello from Nginx container`), output)
	})

	t.Run("check container bind-mounts specs", func(t *testing.T) {
		res := c.RunDockerCmd("inspect", "compose-e2e-volume_nginx_1", "--format", "{{ json .HostConfig.Mounts }}")
		output := res.Stdout()
		// nolint
		assert.Assert(t, strings.Contains(output, `"Type":"bind"`))
		assert.Assert(t, strings.Contains(output, `"Target":"/usr/share/nginx/html"`))
	})

	t.Run("cleanup volume project", func(t *testing.T) {
		c.RunDockerCmd("compose", "--project-name", projectName, "down")
		c.RunDockerCmd("volume", "rm", projectName+"_staticVol")
	})
}
