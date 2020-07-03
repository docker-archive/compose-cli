/*
   Copyright 2020 Docker, Inc.

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

package errors

import (
	"fmt"
	"os"
)

// Error is a CLI error providing the underlying error along with other
// information
type Error struct {
	Err error
	Fix string
}

func (e Error) Error() string {
	return e.Err.Error()
}

// NewImageInaccessibleError returns a CLI error for inaccessible images
func NewImageInaccessibleError(err error) error {
	return Error{
		Err: err,
		Fix: "check the name of the image you specified or try logging into the registry with `docker login`",
	}
}

// Output writes out the error output
func Output(err error) {
	if err == nil {
		return
	}
	if e, ok := err.(Error); ok {
		fmt.Fprintf(os.Stderr, "\n\033[1;31merror\033[0m: %s\n\n", e.Err)
		fmt.Fprintf(os.Stderr, "\033[1;36mfix\033[0m: %s\n", e.Fix)
		return
	}
	fmt.Printf("\033[1;31merror\033[0m: %s\n", err)
}
