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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
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

func TestContextDefault(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	t.Run("show", func(t *testing.T) {
		res := c.RunDockerCmd("context", "show")
		res.Assert(t, icmd.Expected{Out: "default"})
	})

	t.Run("ls", func(t *testing.T) {
		res := c.RunDockerCmd("context", "ls")
		golden.Assert(t, res.Stdout(), GoldenFile("ls-out-default"))

		res = c.RunDockerCmd("context", "ls", "--format", "pretty")
		golden.Assert(t, res.Stdout(), GoldenFile("ls-out-default"))

		res = c.RunDockerCmd("context", "ls", "--format", "json")
		golden.Assert(t, res.Stdout(), GoldenFile("ls-out-json"))

		res = c.RunDockerCmd("context", "ls", "--format", "{{ json . }}")
		golden.Assert(t, res.Stdout(), GoldenFile("ls-out-legacy-json"))
	})

	t.Run("inspect", func(t *testing.T) {
		res := c.RunDockerCmd("context", "inspect", "default")
		res.Assert(t, icmd.Expected{Out: `"Name": "default"`})
	})

	t.Run("inspect current", func(t *testing.T) {
		res := c.RunDockerCmd("context", "inspect")
		res.Assert(t, icmd.Expected{Out: `"Name": "default"`})
	})
}

func TestContextCreateDocker(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)
	res := c.RunDockerCmd("context", "create", "test-docker", "--from", "default")
	res.Assert(t, icmd.Expected{Out: "test-docker"})

	t.Run("ls", func(t *testing.T) {
		res := c.RunDockerCmd("context", "ls")
		golden.Assert(t, res.Stdout(), GoldenFile("ls-out-test-docker"))
	})

	t.Run("ls quiet", func(t *testing.T) {
		res := c.RunDockerCmd("context", "ls", "-q")
		golden.Assert(t, res.Stdout(), "ls-out-test-docker-quiet.golden")
	})

	t.Run("ls format", func(t *testing.T) {
		res := c.RunDockerCmd("context", "ls", "--format", "{{ json . }}")
		res.Assert(t, icmd.Expected{Out: `"Name":"default"`})
	})
}

func TestContextInspect(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)
	res := c.RunDockerCmd("context", "create", "test-docker", "--from", "default")
	res.Assert(t, icmd.Expected{Out: "test-docker"})

	t.Run("inspect current", func(t *testing.T) {
		// Cannot be run in parallel because of "context use"
		res := c.RunDockerCmd("context", "use", "test-docker")
		res.Assert(t, icmd.Expected{Out: "test-docker"})

		res = c.RunDockerCmd("context", "inspect")
		res.Assert(t, icmd.Expected{Out: `"Name": "test-docker"`})
	})
}

func TestContextHelpACI(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	t.Run("help", func(t *testing.T) {
		res := c.RunDockerCmd("context", "create", "aci", "--help")
		// Can't use golden here as the help prints the config directory which changes
		res.Assert(t, icmd.Expected{Out: "docker context create aci CONTEXT [flags]"})
		res.Assert(t, icmd.Expected{Out: "--location"})
		res.Assert(t, icmd.Expected{Out: "--subscription-id"})
		res.Assert(t, icmd.Expected{Out: "--resource-group"})
	})

	t.Run("check exec", func(t *testing.T) {
		res := c.RunDockerOrExitError("context", "create", "aci", "--subscription-id", "invalid-id")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "accepts 1 arg(s), received 0",
		})
		assert.Assert(t, !strings.Contains(res.Combined(), "unknown flag"))
	})
}

func TestContextDuplicateACI(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	c.RunDockerCmd("context", "create", "mycontext", "--from", "default")
	res := c.RunDockerOrExitError("context", "create", "aci", "mycontext")
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "context mycontext: already exists",
	})
}

func TestContextRemove(t *testing.T) {

	t.Run("remove current", func(t *testing.T) {
		c := NewParallelE2eCLI(t, binDir)

		c.RunDockerCmd("context", "create", "test-context-rm", "--from", "default")
		res := c.RunDockerCmd("context", "use", "test-context-rm")
		res.Assert(t, icmd.Expected{Out: "test-context-rm"})
		res = c.RunDockerOrExitError("context", "rm", "test-context-rm")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "cannot delete current context",
		})
	})

	t.Run("force remove current", func(t *testing.T) {
		c := NewParallelE2eCLI(t, binDir)

		c.RunDockerCmd("context", "create", "test-context-rmf")
		c.RunDockerCmd("context", "use", "test-context-rmf")
		res := c.RunDockerCmd("context", "rm", "-f", "test-context-rmf")
		res.Assert(t, icmd.Expected{Out: "test-context-rmf"})
		res = c.RunDockerCmd("context", "ls")
		res.Assert(t, icmd.Expected{Out: "default *"})
	})
}

func TestLoginCommandDelegation(t *testing.T) {
	// These tests just check that the existing CLI is called in various cases.
	// They do not test actual login functionality.
	c := NewParallelE2eCLI(t, binDir)

	t.Run("default context", func(t *testing.T) {
		res := c.RunDockerOrExitError("login", "-u", "nouser", "-p", "wrongpasword")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "unauthorized: incorrect username or password",
		})
	})

	t.Run("interactive", func(t *testing.T) {
		res := c.RunDockerOrExitError("login", "someregistry.docker.io")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "Cannot perform an interactive login from a non TTY device",
		})
	})

	t.Run("localhost registry interactive", func(t *testing.T) {
		res := c.RunDockerOrExitError("login", "localhost:443")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "Cannot perform an interactive login from a non TTY device",
		})
	})

	t.Run("localhost registry", func(t *testing.T) {
		res := c.RunDockerOrExitError("login", "localhost", "-u", "user", "-p", "password")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "http://localhost/v2/",
		})
	})

	t.Run("logout", func(t *testing.T) {
		res := c.RunDockerCmd("logout", "someregistry.docker.io")
		res.Assert(t, icmd.Expected{Out: "Removing login credentials for someregistry.docker.io"})
	})

	t.Run("logout", func(t *testing.T) {
		res := c.RunDockerCmd("logout", "localhost:443")
		res.Assert(t, icmd.Expected{Out: "Removing login credentials for localhost:443"})
	})

	t.Run("existing context", func(t *testing.T) {
		c.RunDockerCmd("context", "create", "local", "local")
		c.RunDockerCmd("context", "use", "local")
		res := c.RunDockerOrExitError("login", "-u", "nouser", "-p", "wrongpasword")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "unauthorized: incorrect username or password",
		})
	})
}

func TestMissingExistingCLI(t *testing.T) {
	t.Parallel()
	home, err := ioutil.TempDir("", "")
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(home)
	})

	bin, err := ioutil.TempDir("", "")
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(bin)
	})
	err = CopyFile(filepath.Join(binDir, DockerExecutableName), filepath.Join(bin, DockerExecutableName))
	assert.NilError(t, err)

	env := []string{"PATH=" + bin}
	if runtime.GOOS == "windows" {
		env = append(env, "USERPROFILE="+home)

	} else {
		env = append(env, "HOME="+home)
	}

	c := icmd.Cmd{
		Env:     env,
		Command: []string{filepath.Join(bin, "docker")},
	}
	res := icmd.RunCmd(c)
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      `"com.docker.cli": executable file not found`,
	})
}

func TestLegacy(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	t.Run("help", func(t *testing.T) {
		res := c.RunDockerCmd("--help")
		res.Assert(t, icmd.Expected{Out: "swarm"})
	})

	t.Run("swarm", func(t *testing.T) {
		res := c.RunDockerOrExitError("swarm", "join")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `"docker swarm join" requires exactly 1 argument.`,
		})
	})

	t.Run("local run", func(t *testing.T) {
		cmd := c.NewDockerCmd("run", "--rm", "hello-world")
		cmd.Timeout = 40 * time.Second
		res := icmd.RunCmd(cmd)
		res.Assert(t, icmd.Expected{Out: "Hello from Docker!"})
	})

	t.Run("error messages", func(t *testing.T) {
		res := c.RunDockerOrExitError("foo")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "docker: 'foo' is not a docker command.",
		})
	})

	t.Run("run without HOME defined", func(t *testing.T) {
		cmd := c.NewDockerCmd("ps")
		cmd.Env = []string{"PATH=" + c.PathEnvVar()}
		res := icmd.RunCmd(cmd)
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "CONTAINER ID",
		})
		assert.Equal(t, res.Stderr(), "")
	})

	t.Run("run without write access to context store", func(t *testing.T) {
		cmd := c.NewDockerCmd("ps")
		cmd.Env = []string{"PATH=" + c.PathEnvVar(), "HOME=/doesnotexist/"}
		res := icmd.RunCmd(cmd)
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "CONTAINER ID",
		})
	})

	t.Run("host flag", func(t *testing.T) {
		stderr := "Cannot connect to the Docker daemon at tcp://localhost:123"
		if runtime.GOOS == "windows" {
			stderr = "error during connect: Get http://localhost:123"
		}
		res := c.RunDockerOrExitError("-H", "tcp://localhost:123", "version")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      stderr,
		})
	})

	t.Run("existing contexts delegate", func(t *testing.T) {
		c.RunDockerCmd("context", "create", "moby-ctx", "--from=default")
		c.RunDockerCmd("context", "use", "moby-ctx")
		res := c.RunDockerOrExitError("swarm", "join")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `"docker swarm join" requires exactly 1 argument.`,
		})
		res = c.RunDockerCmd("context", "update", "moby-ctx", "--description", "updated context")
		res.Assert(t, icmd.Expected{Out: "moby-ctx"})
	})

	t.Run("host flag overrides context", func(t *testing.T) {
		c.RunDockerCmd("context", "create", "local", "test-local")
		c.RunDockerCmd("context", "use", "test-local")
		endpoint := "unix:///var/run/docker.sock"
		if runtime.GOOS == "windows" {
			endpoint = "npipe:////./pipe/docker_engine"
		}
		res := c.RunDockerCmd("-H", endpoint, "images")
		// Local backend does not have images command
		assert.Assert(t, strings.Contains(res.Stdout(), "IMAGE ID"), res.Stdout())
	})
}

func TestLegacyLogin(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	t.Run("host flag login", func(t *testing.T) {
		res := c.RunDockerOrExitError("-H", "tcp://localhost:123", "login", "-u", "nouser", "-p", "wrongpasword")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "WARNING! Using --password via the CLI is insecure. Use --password-stdin.",
		})
	})

	t.Run("log level flag login", func(t *testing.T) {
		res := c.RunDockerOrExitError("--log-level", "debug", "login", "-u", "nouser", "-p", "wrongpasword")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "WARNING! Using --password via the CLI is insecure",
		})
	})

	t.Run("login help global flags", func(t *testing.T) {
		res := c.RunDockerCmd("login", "--help")
		assert.Assert(t, !strings.Contains(res.Combined(), "--log-level"))
	})
}

func TestUnsupportedCommand(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	c.RunDockerCmd("context", "create", "local", "test-local")
	res := c.RunDockerOrExitError("--context", "test-local", "images")
	res.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      `Command "images" not available in current context (test-local), you can use the "default" context to run this command`,
	})
}

func TestBackendMetadata(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	t.Run("backend-metadata", func(t *testing.T) {
		res := c.RunDockerCmd("backend-metadata")
		res.Assert(t, icmd.Expected{Out: `{"Name":"Cloud integration","Version":"`})
	})
}

func TestFailOnEcsUsageAsPlugin(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)
	res := c.RunDockerCmd("context", "create", "local", "local")
	res.Assert(t, icmd.Expected{})

	t.Run("fail on ecs usage as plugin", func(t *testing.T) {
		res := c.RunDockerOrExitError("--context", "local", "ecs", "compose", "up")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Out:      "",
			Err:      "The ECS integration is now part of the CLI. Use `docker compose` with an ECS context.",
		})
	})
}
