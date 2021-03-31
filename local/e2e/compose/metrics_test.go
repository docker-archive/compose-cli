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
	"fmt"
	"syscall"
	"testing"
	"time"

	"gotest.tools/assert"
	"gotest.tools/v3/icmd"

	. "github.com/docker/compose-cli/utils/e2e"
)

func TestComposeMetrics(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)
	s := NewMetricsServer(c.MetricsSocket())
	s.Start()
	defer s.Stop()

	started := false
	for i := 0; i < 30; i++ {
		c.RunDockerCmd("help", "ps")
		if len(s.GetUsage()) > 0 {
			started = true
			fmt.Printf("	[%s] Server up in %d ms\n", t.Name(), i*100)
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.Assert(t, started, "Metrics mock server not available after 3 secs")

	t.Run("metrics on Compose commands", func(t *testing.T) {
		s.ResetUsage()

		c.RunDockerCmd("compose", "ls")

		upProjectPath := "../compose/fixtures/simple-composefile/compose.yml"
		cmd := c.NewDockerCmd("compose", "-f", upProjectPath, "up")
		defer c.RunDockerCmd("compose", "-f", upProjectPath, "down")
		upProcess := icmd.StartCmd(cmd)
		c.WaitForOutputResult(upProcess, StdoutContains("CPU:"), 10*time.Second, 1*time.Second)

		res := c.RunDockerOrExitError("compose", "-f", upProjectPath, "ps", "--all")
		res.Assert(t, icmd.Expected{Out: "running"})

		err := upProcess.Cmd.Process.Signal(syscall.SIGINT)
		assert.NilError(t, err, upProcess.Combined())
		c.WaitForOutputResult(upProcess, StdoutContains("Gracefully stopping..."), 10*time.Second, 1*time.Second)

		c.RunDockerOrExitError("compose", "ps")

		usage := s.GetUsage()
		assert.DeepEqual(t, []string{
			`{"command":"compose ls","context":"moby","source":"cli","status":"success"}`,
			`{"command":"compose ps","context":"moby","source":"cli","status":"success"}`,
			`{"command":"compose up","context":"moby","source":"cli","status":"success"}`,
			`{"command":"compose ps","context":"moby","source":"cli","status":"failure"}`,
		}, usage)
	})

	t.Run("metrics on cancel Compose build", func(t *testing.T) {
		s.ResetUsage()

		buildProjectPath := "../compose/fixtures/build-infinite/compose.yml"
		cmd := c.NewDockerCmd("compose", "-f", buildProjectPath, "build")
		buildProcess := icmd.StartCmd(cmd)
		c.WaitForOutputResult(buildProcess, StdoutContains("RUN sleep infinity"), 10*time.Second, 1*time.Second)

		err := buildProcess.Cmd.Process.Signal(syscall.SIGINT)
		assert.NilError(t, err, buildProcess.Combined())
		c.WaitForOutputResult(buildProcess, StdoutContains("CANCELED"), 10*time.Second, 1*time.Second)

		usage := s.GetUsage()
		assert.DeepEqual(t, []string{
			`{"command":"compose build","context":"moby","source":"cli","status":"canceled"}`,
		}, usage)
	})
}
