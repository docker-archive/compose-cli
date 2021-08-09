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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/compose-cli/utils"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"

	"github.com/docker/compose-cli/pkg/api"
	"github.com/docker/compose-cli/pkg/progress"
)

func (b *ecsAPIService) Up(ctx context.Context, project *types.Project, options api.UpOptions) error {
	if err := checkUnSupportedUpOptions(options); err != nil {
		return err
	}
	return progress.Run(ctx, func(ctx context.Context) error {
		return b.up(ctx, project, options)
	})
}

func (b *ecsAPIService) up(ctx context.Context, project *types.Project, options api.UpOptions) error {
	logrus.Debugf("deploying on AWS with region=%q", b.Region)
	err := b.aws.CheckRequirements(ctx, b.Region)
	if err != nil {
		return err
	}

	template, err := b.Convert(ctx, project, api.ConvertOptions{
		Format: "yaml",
	})
	if err != nil {
		return err
	}

	update, err := b.aws.StackExists(ctx, project.Name)
	if err != nil {
		return err
	}

	var previousEvents []string
	if update {
		var err error
		previousEvents, err = b.previousStackEvents(ctx, project.Name)
		if err != nil {
			return err
		}
	}

	operation := stackCreate
	if update {
		operation = stackUpdate
		changeset, err := b.aws.CreateChangeSet(ctx, project.Name, b.Region, template)
		if err != nil {
			return err
		}
		err = b.aws.UpdateStack(ctx, changeset)
		if err != nil {
			return err
		}
	} else {
		err = b.aws.CreateStack(ctx, project.Name, b.Region, template)
		if err != nil {
			return err
		}
	}
	if options.Start.Attach == nil {
		return nil
	}
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		fmt.Println("user interrupted deployment. Deleting stack...")
		b.Down(ctx, project.Name, api.DownOptions{}) // nolint:errcheck
	}()

	err = b.WaitStackCompletion(ctx, project.Name, operation, previousEvents...)
	return err
}

func checkUnSupportedUpOptions(o api.UpOptions) error {
	var errs *multierror.Error
	errs = utils.CheckUnsupported(errs, o.Create.Inherit, false, "--renew-anon-volumes")
	errs = utils.CheckUnsupported(errs, o.Create.RemoveOrphans, false, "--remove-orphans")
	errs = utils.CheckUnsupported(errs, o.Create.QuietPull, false, "--quiet-pull")
	errs = utils.CheckUnsupported(errs, o.Create.Recreate, "", "--force-recreate")
	errs = utils.CheckUnsupported(errs, o.Create.RecreateDependencies, "", "--always-recreate-deps")
	errs = utils.CheckUnsupportedDurationPtr(errs, o.Create.Timeout, nil, "--timeout")
	errs = utils.CheckUnsupported(errs, len(o.Start.AttachTo), 0, "--attach-dependencies")
	errs = utils.CheckUnsupported(errs, len(o.Start.ExitCodeFrom), 0, "--exit-code-from")
	return errs.ErrorOrNil()
}
