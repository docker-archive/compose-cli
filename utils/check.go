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

package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/compose-cli/api/context/store"
	"github.com/docker/compose-cli/cli/config"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

var ErrUnsupportedFlag = errors.New("unsupported flag")

func CheckUnsupported(errs error, toCheck, expectedValue interface{}, commandName, msg string) error {
	ctype := store.DefaultContextType
	currentContext := config.GetCurrentContext("", config.ConfDir(), nil)
	s, err := store.New(config.ConfDir())
	if err != nil {
		return err
	}
	cc, err := s.Get(currentContext)
	if err != nil {
		return err
	}
	if cc != nil {
		ctype = cc.Type()
	}
	// Fixes type for posterior comparison
	switch toCheck.(type) {
	case *time.Duration:
		if expectedValue == nil {
			var nilDurationPtr *time.Duration
			expectedValue = nilDurationPtr
		}
	}
	if toCheck != expectedValue {
		return multierror.Append(errs, errors.Wrap(ErrUnsupportedFlag,
			fmt.Sprintf(`option "%s --%s" on context type %s.`,
				commandName, msg, strings.ToUpper(ctype))))
	}
	return errs
}
