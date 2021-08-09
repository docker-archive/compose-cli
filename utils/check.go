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
	"time"

	"github.com/docker/compose-cli/pkg/api"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

func CheckUnsupportedDurationPtr(errs *multierror.Error, toCheck, expectedValue interface{}, msg string) *multierror.Error {
	var nilDurationPtr *time.Duration
	if expectedValue == nil {
		expectedValue = nilDurationPtr
	}
	return CheckUnsupported(errs, toCheck, expectedValue, msg)
}

func CheckUnsupportedStringSlicePtr(errs *multierror.Error, toCheck, expectedValue interface{}, msg string) *multierror.Error {
	var nilStringSlicePtr *[]string
	if expectedValue == nil {
		expectedValue = nilStringSlicePtr
	}
	return CheckUnsupported(errs, toCheck, expectedValue, msg)
}

func CheckUnsupported(errs *multierror.Error, toCheck, expectedValue interface{}, msg string) *multierror.Error {
	if toCheck != expectedValue {
		return multierror.Append(errs, errors.Wrap(api.ErrUnSupported, msg+" option is not supported on ECS"))
	}
	return errs
}
