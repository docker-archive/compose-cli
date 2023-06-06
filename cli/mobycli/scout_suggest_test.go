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
	"testing"

	"gotest.tools/v3/assert"
)

func TestHintsEnabled(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func()
		expected bool
	}{
		{
			"enabled by default",
			func() {},
			true,
		},
		{
			"handle true value",
			func() {
				t.Setenv(scoutHintEnvVarName, "t")
			},
			true,
		},
		{
			"handle false value",
			func() {
				t.Setenv(scoutHintEnvVarName, "FALSE")
			},
			false,
		},
		{
			"handle error",
			func() {
				t.Setenv(scoutHintEnvVarName, "123")
			},
			true,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			assert.Equal(t, isDockerScoutHintsEnabled(), tc.expected)
		})
	}
}
