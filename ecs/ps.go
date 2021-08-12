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

func (b *ecsAPIService) Ps(ctx context.Context, projectName string, options api.PsOptions) ([]api.ContainerSummary, error) {
	if err := checkUnsupportedPsOptions(options); err != nil {
		return nil, err
	}

	cluster, err := b.aws.GetStackClusterID(ctx, projectName)
	if err != nil {
		return nil, err
	}
	servicesARN, err := b.aws.ListStackServices(ctx, projectName)
	if err != nil {
		return nil, err
	}

	if len(servicesARN) == 0 {
		return nil, nil
	}

	summary := []api.ContainerSummary{}
	for _, arn := range servicesARN {
		service, err := b.aws.DescribeService(ctx, cluster, arn)
		if err != nil {
			return nil, err
		}

		tasks, err := b.aws.DescribeServiceTasks(ctx, cluster, projectName, service.Name)
		if err != nil {
			return nil, err
		}

		for i, t := range tasks {
			t.Publishers = service.Publishers
			tasks[i] = t
		}
		summary = append(summary, tasks...)
	}
	return summary, nil
}

func checkUnsupportedPsOptions(o api.PsOptions) error {
	var errs *multierror.Error
	errs = utils.CheckUnsupported(errs, o.All, false, "ps", "all")
	return errs.ErrorOrNil()
}
