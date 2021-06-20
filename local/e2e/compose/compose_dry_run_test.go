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
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	. "github.com/docker/compose-cli/utils/e2e"
)

func TestComposePullDryRun(t *testing.T) {
	c := NewParallelE2eCLI(t, binDir)

	t.Run("compose pull dry run", func(t *testing.T) {
		// ensure storing alpine image and deleting hello-world
		c.RunDockerCmd("pull", "alpine")
		c.RunDockerCmd("rmi", "hello-world", "-f")

		res := c.RunDockerCmd("compose", "-f", "./fixtures/dry-run-test/pull/compose.yaml", "pull", "--dry-run")
		lines := Lines(res.Stdout())
		for _, line := range lines {
			if strings.Contains(line, "expected-skip") {
				assert.Assert(t, strings.Contains(line, " skip "))
			}
			if strings.Contains(line, "expected-fail") {
				assert.Assert(t, strings.Contains(line, " fail "))
			}
			if strings.Contains(line, "expected-fetch") {
				assert.Assert(t, strings.Contains(line, " fetch "))
			}
		}
	})

}
