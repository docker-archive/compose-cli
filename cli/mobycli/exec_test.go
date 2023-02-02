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
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/stretchr/testify/require"

	"github.com/docker/compose-cli/api/context/store"
)

func TestDelegateContextTypeToMoby(t *testing.T) {

	isDelegated := func(val string) bool {
		for _, ctx := range delegatedContextTypes {
			if ctx == val {
				return true
			}
		}
		return false
	}

	allCtx := []string{store.AciContextType, store.EcsContextType, store.AwsContextType, store.DefaultContextType}
	for _, ctx := range allCtx {
		if isDelegated(ctx) {
			assert.Assert(t, mustDelegateToMoby(ctx))
			continue
		}
		assert.Assert(t, !mustDelegateToMoby(ctx))
	}
}

func TestFindComDockerCli(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "bin")

	// com.docker.cli on path
	pathFile := filepath.Join(bin, ComDockerCli)
	require.NoError(t, os.MkdirAll(bin, os.ModePerm))
	require.NoError(t, os.WriteFile(pathFile, []byte(""), os.ModePerm))

	exe, err := os.Executable()
	require.NoError(t, err)
	exe, err = filepath.EvalSymlinks(exe)
	require.NoError(t, err)

	// com.docker.cli in same directory as current executable
	currFile := filepath.Join(filepath.Dir(exe), ComDockerCli)
	require.NoError(t, os.WriteFile(currFile, []byte(""), os.ModePerm))

	savePath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", savePath) })

	_ = os.Setenv("PATH", bin)

	t.Run("from $DOCKER_COM_DOCKER_CLI", func(t *testing.T) {
		fromenv := filepath.Join(tmp, "fromenv")
		_ = os.Setenv("DOCKER_COM_DOCKER_CLI", fromenv)
		t.Cleanup(func() { _ = os.Unsetenv("DOCKER_COM_DOCKER_CLI") })

		assert.Equal(t, fromenv, comDockerCli())
	})

	t.Run("from binary next to current executable", func(t *testing.T) {
		assert.Equal(t, currFile, comDockerCli())
	})

	t.Run("from binary on path", func(t *testing.T) {
		_ = os.Remove(currFile)
		assert.Equal(t, pathFile, comDockerCli())
	})
}
