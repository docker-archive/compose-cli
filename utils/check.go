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
	"context"
	"fmt"
	"strings"

	"github.com/docker/compose/v2/pkg/api"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/docker/compose-cli/api/config"
)

// CheckUnsupported checks if a flag was used when it shouldn't and adds an error in case
func CheckUnsupported(ctx context.Context, errs error, toCheck, expectedValue interface{}, commandName, msg string) error {
	if !(isNil(toCheck) && isNil(expectedValue)) && toCheck != expectedValue {
		ctype := ctx.Value(config.ContextTypeKey).(string)
		return multierror.Append(errs, errors.Wrap(api.ErrUnsupportedFlag,
			fmt.Sprintf(`option "%s --%s" on context type %s.`,
				commandName, msg, strings.ToUpper(ctype))))
	}
	return errs
}
