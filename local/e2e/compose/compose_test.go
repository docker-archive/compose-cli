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

package e2e

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"

	. "github.com/docker/compose-cli/utils/e2e"
)

var binDir string

func TestMain(m *testing.M) {
	p, cleanup, err := SetupExistingCLI()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	binDir = p
	exitCode := m.Run()
	cleanup()
	os.Exit(exitCode)
}

func TestLocalComposeUp(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	const projectName = "compose-e2e-demo"

	t.Run("up", func(t *testing.T) {
		c.RunDockerCmd("compose", "-f", "./fixtures/sentences/compose.yaml", "--project-name", projectName, "up", "-d")
	})

	t.Run("check accessing running app", func(t *testing.T) {
		res := c.RunDockerCmd("compose", "-p", projectName, "ps")
		res.Assert(t, icmd.Expected{Out: `web`})

		endpoint := "http://localhost:90"
		output := HTTPGetWithRetry(t, endpoint+"/words/noun", http.StatusOK, 2*time.Second, 20*time.Second)
		assert.Assert(t, strings.Contains(output, `"word":`))

		res = c.RunDockerCmd("network", "ls")
		res.Assert(t, icmd.Expected{Out: projectName + "_default"})
	})

	t.Run("check compose labels", func(t *testing.T) {
		wd, _ := os.Getwd()
		res := c.RunDockerCmd("inspect", projectName+"_web_1")
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.container-number": "1"`})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.project": "compose-e2e-demo"`})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.oneoff": "False",`})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.config-hash":`})
		res.Assert(t, icmd.Expected{Out: fmt.Sprintf(`"com.docker.compose.project.config_files": "%s/fixtures/sentences/compose.yaml"`, wd)})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.project.working_dir":`})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.service": "web"`})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.version":`})

		res = c.RunDockerCmd("network", "inspect", projectName+"_default")
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.network": "default"`})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.project": `})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.version": `})
	})

	t.Run("check user labels", func(t *testing.T) {
		res := c.RunDockerCmd("inspect", projectName+"_web_1")
		res.Assert(t, icmd.Expected{Out: `"my-label": "test"`})

	})

	t.Run("check healthcheck output", func(t *testing.T) {
		c.WaitForCmdResult(c.NewDockerCmd("compose", "-p", projectName, "ps", "--format", "json"),
			StdoutContains(`"Name":"compose-e2e-demo_web_1","Project":"compose-e2e-demo","Service":"web","State":"running","Health":"healthy"`),
			5*time.Second, 1*time.Second)

		res := c.RunDockerCmd("compose", "-p", projectName, "ps")
		res.Assert(t, icmd.Expected{Out: `NAME                       SERVICE             STATUS              PORTS`})
		res.Assert(t, icmd.Expected{Out: `compose-e2e-demo_web_1     web                 running (healthy)   0.0.0.0:90->80/tcp`})
		res.Assert(t, icmd.Expected{Out: `compose-e2e-demo_db_1      db                  running             5432/tcp`})
	})

	t.Run("down", func(t *testing.T) {
		_ = c.RunDockerCmd("compose", "--project-name", projectName, "down")
	})

	t.Run("check containers after down", func(t *testing.T) {
		res := c.RunDockerCmd("ps", "--all")
		assert.Assert(t, !strings.Contains(res.Combined(), projectName), res.Combined())
	})

	t.Run("check networks after down", func(t *testing.T) {
		res := c.RunDockerCmd("network", "ls")
		assert.Assert(t, !strings.Contains(res.Combined(), projectName), res.Combined())
	})
}

func TestLocalComposeRun(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	t.Run("compose run", func(t *testing.T) {
		res := c.RunDockerCmd("compose", "-f", "./fixtures/run-test/compose.yml", "run", "back")
		lines := Lines(res.Stdout())
		assert.Equal(t, lines[len(lines)-1], "Hello there!!", res.Stdout())
		assert.Assert(t, !strings.Contains(res.Combined(), "orphan"))

		res = c.RunDockerCmd("compose", "-f", "./fixtures/run-test/compose.yml", "run", "back", "echo", "Hello one more time")
		lines = Lines(res.Stdout())
		assert.Equal(t, lines[len(lines)-1], "Hello one more time", res.Stdout())
		assert.Assert(t, !strings.Contains(res.Combined(), "orphan"))
	})

	t.Run("check run container exited", func(t *testing.T) {
		res := c.RunDockerCmd("ps", "--all")
		lines := Lines(res.Stdout())
		var runContainerID string
		var truncatedSlug string
		for _, line := range lines {
			fields := strings.Fields(line)
			containerID := fields[len(fields)-1]
			assert.Assert(t, !strings.HasPrefix(containerID, "run-test_front"))
			if strings.HasPrefix(containerID, "run-test_back") {
				// only the one-off container for back service
				assert.Assert(t, strings.HasPrefix(containerID, "run-test_back_run_"), containerID)
				truncatedSlug = strings.Replace(containerID, "run-test_back_run_", "", 1)
				runContainerID = containerID
			}
			if strings.HasPrefix(containerID, "run-test_db_1") {
				assert.Assert(t, strings.Contains(line, "Up"), line)
			}
		}
		assert.Assert(t, runContainerID != "")
		res = c.RunDockerCmd("inspect", runContainerID)
		res.Assert(t, icmd.Expected{Out: ` "Status": "exited"`})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.container-number": "1"`})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.project": "run-test"`})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.oneoff": "True",`})
		res.Assert(t, icmd.Expected{Out: `"com.docker.compose.slug": "` + truncatedSlug})
	})

	t.Run("compose run --rm", func(t *testing.T) {
		res := c.RunDockerCmd("compose", "-f", "./fixtures/run-test/compose.yml", "run", "--rm", "back", "/bin/sh", "-c", "echo Hello again")
		lines := Lines(res.Stdout())
		assert.Equal(t, lines[len(lines)-1], "Hello again", res.Stdout())

		res = c.RunDockerCmd("ps", "--all")
		assert.Assert(t, strings.Contains(res.Stdout(), "run-test_back"), res.Stdout())
	})

	t.Run("down", func(t *testing.T) {
		c.RunDockerCmd("compose", "-f", "./fixtures/run-test/compose.yml", "down")
		res := c.RunDockerCmd("ps", "--all")
		assert.Assert(t, !strings.Contains(res.Stdout(), "run-test"), res.Stdout())
	})
}

func TestNetworks(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	const projectName = "network_e2e"

	t.Run("ensure we do not reuse previous networks", func(t *testing.T) {
		c.RunDockerOrExitError("network", "rm", projectName+"_dbnet")
		c.RunDockerOrExitError("network", "rm", "microservices")
	})

	t.Run("up", func(t *testing.T) {
		c.RunDockerCmd("compose", "-f", "./fixtures/network-test/compose.yaml", "--project-name", projectName, "up", "-d")
	})

	t.Run("check running project", func(t *testing.T) {
		res := c.RunDockerCmd("compose", "-p", projectName, "ps")
		res.Assert(t, icmd.Expected{Out: `web`})

		endpoint := "http://localhost:80"
		output := HTTPGetWithRetry(t, endpoint+"/words/noun", http.StatusOK, 2*time.Second, 20*time.Second)
		assert.Assert(t, strings.Contains(output, `"word":`))

		res = c.RunDockerCmd("network", "ls")
		res.Assert(t, icmd.Expected{Out: projectName + "_dbnet"})
		res.Assert(t, icmd.Expected{Out: "microservices"})
	})

	t.Run("down", func(t *testing.T) {
		_ = c.RunDockerCmd("compose", "--project-name", projectName, "down")
	})

	t.Run("check networks after down", func(t *testing.T) {
		res := c.RunDockerCmd("network", "ls")
		assert.Assert(t, !strings.Contains(res.Combined(), projectName), res.Combined())
		assert.Assert(t, !strings.Contains(res.Combined(), "microservices"), res.Combined())
	})
}

func TestLocalComposeBuild(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	t.Run("build named and unnamed images", func(t *testing.T) {
		// ensure local test run does not reuse previously build image
		c.RunDockerOrExitError("rmi", "build-test_nginx")
		c.RunDockerOrExitError("rmi", "custom-nginx")

		res := c.RunDockerCmd("compose", "--workdir", "fixtures/build-test", "build")

		res.Assert(t, icmd.Expected{Out: "COPY static /usr/share/nginx/html"})
		c.RunDockerCmd("image", "inspect", "build-test_nginx")
		c.RunDockerCmd("image", "inspect", "custom-nginx")
	})

	t.Run("build as part of up", func(t *testing.T) {
		c.RunDockerOrExitError("rmi", "build-test_nginx")
		c.RunDockerOrExitError("rmi", "custom-nginx")

		res := c.RunDockerCmd("compose", "--workdir", "fixtures/build-test", "up", "-d")
		t.Cleanup(func() {
			c.RunDockerCmd("compose", "--workdir", "fixtures/build-test", "down")
		})

		res.Assert(t, icmd.Expected{Out: "COPY static /usr/share/nginx/html"})

		output := HTTPGetWithRetry(t, "http://localhost:8070", http.StatusOK, 2*time.Second, 20*time.Second)
		assert.Assert(t, strings.Contains(output, "Hello from Nginx container"))

		c.RunDockerCmd("image", "inspect", "build-test_nginx")
		c.RunDockerCmd("image", "inspect", "custom-nginx")
	})

	t.Run("no rebuild when up again", func(t *testing.T) {
		res := c.RunDockerCmd("compose", "--workdir", "fixtures/build-test", "up", "-d")

		assert.Assert(t, !strings.Contains(res.Stdout(), "COPY static /usr/share/nginx/html"), res.Stdout())
	})

	t.Run("cleanup build project", func(t *testing.T) {
		c.RunDockerCmd("compose", "--workdir", "fixtures/build-test", "down")
		c.RunDockerCmd("rmi", "build-test_nginx")
		c.RunDockerCmd("rmi", "custom-nginx")
	})
}

func TestLocalComposeVolume(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	const projectName = "compose-e2e-volume"

	t.Run("up with build and no image name, volume", func(t *testing.T) {
		// ensure local test run does not reuse previously build image
		c.RunDockerOrExitError("rmi", "compose-e2e-volume_nginx")
		c.RunDockerOrExitError("volume", "rm", projectName+"_staticVol")
		c.RunDockerOrExitError("volume", "rm", "myvolume")
		c.RunDockerCmd("compose", "--workdir", "fixtures/volume-test", "--project-name", projectName, "up", "-d")
	})

	t.Run("access bind mount data", func(t *testing.T) {
		output := HTTPGetWithRetry(t, "http://localhost:8090", http.StatusOK, 2*time.Second, 20*time.Second)
		assert.Assert(t, strings.Contains(output, "Hello from Nginx container"))
	})

	t.Run("check container volume specs", func(t *testing.T) {
		res := c.RunDockerCmd("inspect", "compose-e2e-volume_nginx2_1", "--format", "{{ json .Mounts }}")
		output := res.Stdout()
		// nolint
		assert.Assert(t, strings.Contains(output, `"Destination":"/usr/src/app/node_modules","Driver":"local","Mode":"","RW":true,"Propagation":""`), output)
		assert.Assert(t, strings.Contains(output, `"Destination":"/myconfig","Mode":"","RW":false,"Propagation":"rprivate"`), output)
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

func TestComposePull(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	res := c.RunDockerOrExitError("compose", "--workdir", "fixtures/simple-composefile", "pull")
	output := res.Combined()

	assert.Assert(t, strings.Contains(output, "simple Pulled"))
	assert.Assert(t, strings.Contains(output, "another Pulled"))
}
