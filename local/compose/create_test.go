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

package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/types"
	composetypes "github.com/compose-spec/compose-go/types"
	mountTypes "github.com/docker/docker/api/types/mount"
	"gotest.tools/v3/assert"
)

func TestBuildBindMount(t *testing.T) {
	project := composetypes.Project{}
	volume := composetypes.ServiceVolumeConfig{
		Type:   composetypes.VolumeTypeBind,
		Source: "",
		Target: "/data",
	}
	mount, err := buildMount(project, volume)
	assert.NilError(t, err)
	assert.Assert(t, filepath.IsAbs(mount.Source))
	_, err = os.Stat(mount.Source)
	assert.NilError(t, err)
	assert.Equal(t, mount.Type, mountTypes.TypeBind)
}

func TestBuildVolumeMount(t *testing.T) {
	project := composetypes.Project{
		Name: "myProject",
		Volumes: composetypes.Volumes(map[string]composetypes.VolumeConfig{
			"myVolume": {
				Name: "myProject_myVolume",
			},
		}),
	}
	volume := composetypes.ServiceVolumeConfig{
		Type:   composetypes.VolumeTypeVolume,
		Source: "myVolume",
		Target: "/data",
	}
	mount, err := buildMount(project, volume)
	assert.NilError(t, err)
	assert.Equal(t, mount.Source, "myProject_myVolume")
	assert.Equal(t, mount.Type, mountTypes.TypeVolume)
}

func TestServiceImageName(t *testing.T) {
	assert.Equal(t, getImageName(types.ServiceConfig{Image: "myImage"}, "myProject"), "myImage")
	assert.Equal(t, getImageName(types.ServiceConfig{Name: "aService"}, "myProject"), "myProject_aService")
}
