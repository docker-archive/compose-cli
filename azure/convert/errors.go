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

package convert

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/containerinstance/mgmt/containerinstance"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/pkg/errors"

	"github.com/docker/api/errdefs"
)

const (
	// CodeInaccessibleImage is returned by Azure when a user references an
	// image that does not exist or is inaccessible
	CodeInaccessibleImage = "InaccessibleImage"
)

// FromAzureError checks if an error is from the Azure SDK and changes it to one
// of our API errors
func FromAzureError(e error, group containerinstance.ContainerGroup) error {
	if e == nil {
		return nil
	}
	var ed autorest.DetailedError
	if !errors.As(e, &ed) {
		return e
	}
	var es *azure.ServiceError
	if !errors.As(ed.Original, &es) {
		return ed
	}
	return fromServiceError(es, group)
}

func fromServiceError(e *azure.ServiceError, group containerinstance.ContainerGroup) error {
	switch e.Code {
	case CodeInaccessibleImage:
		image := imageFromError(e, group)
		if image == "" {
			return errors.Wrap(errdefs.ErrImageInaccessible, "unable to get image")
		}
		return errors.Wrapf(errdefs.ErrImageInaccessible, "unable to access %q", image)
	default:
		return errors.Wrap(errdefs.ErrUnknown, e.Error())
	}
}

func imageFromError(e *azure.ServiceError, group containerinstance.ContainerGroup) string {
	for _, c := range *group.Containers {
		if strings.Contains(e.Message, *c.Image) {
			return *c.Image
		}
	}
	return ""
}
