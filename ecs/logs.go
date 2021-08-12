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

package ecs

import (
	"context"

	"github.com/docker/compose-cli/pkg/api"
	"github.com/docker/compose-cli/utils"
	"github.com/hashicorp/go-multierror"
)

func (b *ecsAPIService) Logs(ctx context.Context, projectName string, consumer api.LogConsumer, options api.LogOptions) error {
	if err := checkUnsupportedLogOptions(options); err != nil {
		return err
	}
	if len(options.Services) > 0 {
		consumer = utils.FilteredLogConsumer(consumer, options.Services)
	}
	err := b.aws.GetLogs(ctx, projectName, consumer.Log, options.Follow)
	return err
}

func checkUnsupportedLogOptions(o api.LogOptions) error {
	var errs *multierror.Error
	checks := []struct {
		toCheck, expected interface{}
		option            string
	}{
		{o.Since, "", "since"},
		{o.Tail, "", "tail"},
		{o.Timestamps, false, "timestamps"},
		{o.Until, "", "until"},
	}
	for _, c := range checks {
		errs = utils.CheckUnsupported(errs, c.toCheck, c.expected, "logs", c.option)
	}
	return errs.ErrorOrNil()
}
