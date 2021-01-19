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

package formatter

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/docker/compose-cli/api/containers"
)

func TestDisplayPortsNoDomainname(t *testing.T) {

	testCases := []struct {
		name     string
		in       []containers.Port
		expected []string
	}{
		{
			name:     "simple",
			in:       []containers.Port{{HostPort: 80, ContainerPort: 80, Protocol: "tcp"}},
			expected: []string{"0.0.0.0:80->80/tcp"},
		},
		{
			name:     "different ports",
			in:       []containers.Port{{HostPort: 80, ContainerPort: 90, Protocol: "tcp"}},
			expected: []string{"0.0.0.0:80->90/tcp"},
		},
		{
			name:     "host ip",
			in:       []containers.Port{{HostIP: "192.168.0.1", HostPort: 80, ContainerPort: 90, Protocol: "tcp"}},
			expected: []string{"192.168.0.1:80->90/tcp"},
		},
		{
			name:     "grouping",
			in:       []containers.Port{{HostPort: 80, ContainerPort: 80, Protocol: "tcp"}, {HostPort: 81, ContainerPort: 81, Protocol: "tcp"}},
			expected: []string{"0.0.0.0:80-81->80-81/tcp"},
		},
		{
			name:     "groups",
			in:       []containers.Port{{HostPort: 80, ContainerPort: 80, Protocol: "tcp"}, {HostPort: 82, ContainerPort: 82, Protocol: "tcp"}},
			expected: []string{"0.0.0.0:80->80/tcp", "0.0.0.0:82->82/tcp"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			out := PortsToStrings(testCase.in, "")
			assert.DeepEqual(t, testCase.expected, out)
		})
	}
}

func TestDisplayPortsWithDomainname(t *testing.T) {
	containerConfig := containers.ContainerConfig{
		Ports: []containers.Port{
			{
				HostPort:      80,
				ContainerPort: 80,
				Protocol:      "tcp",
			},
		},
	}

	out := PortsToStrings(containerConfig.Ports, "mydomain.westus.azurecontainner.io")
	assert.DeepEqual(t, []string{"mydomain.westus.azurecontainner.io:80->80/tcp"}, out)
}
