/*
   Copyright 2021 Docker Compose CLI authors

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

package login

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/2019-03-01/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/subscription/mgmt/subscription"
	"github.com/Azure/azure-sdk-for-go/services/containerinstance/mgmt/2019-12-01/containerinstance"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/pkg/errors"

	"github.com/docker/compose-cli/api/errdefs"
	"github.com/docker/compose-cli/internal"
)

// NewContainerGroupsClient get client toi manipulate containerGrouos
func NewContainerGroupsClient(subscriptionID string) (containerinstance.ContainerGroupsClient, error) {
	containerGroupsClient := containerinstance.NewContainerGroupsClient(subscriptionID)
	err := setupClient(&containerGroupsClient.Client)
	if err != nil {
		return containerinstance.ContainerGroupsClient{}, err
	}
	containerGroupsClient.PollingDelay = 5 * time.Second
	containerGroupsClient.RetryAttempts = 30
	containerGroupsClient.RetryDuration = 1 * time.Second
	return containerGroupsClient, nil
}

func setupClient(aciClient *autorest.Client) error {
	aciClient.UserAgent = internal.UserAgentName + "/" + internal.Version
	auth, err := NewAuthorizerFromLogin()
	if err != nil {
		return err
	}
	aciClient.Authorizer = auth
	return nil
}

// NewStorageAccountsClient get client to manipulate storage accounts
func NewStorageAccountsClient(subscriptionID string) (storage.AccountsClient, error) {
	containerGroupsClient := storage.NewAccountsClient(subscriptionID)
	err := setupClient(&containerGroupsClient.Client)
	if err != nil {
		return storage.AccountsClient{}, err
	}
	containerGroupsClient.PollingDelay = 5 * time.Second
	containerGroupsClient.RetryAttempts = 30
	containerGroupsClient.RetryDuration = 1 * time.Second
	return containerGroupsClient, nil
}

// NewFileShareClient get client to manipulate file shares
func NewFileShareClient(subscriptionID string) (storage.FileSharesClient, error) {
	containerGroupsClient := storage.NewFileSharesClient(subscriptionID)
	err := setupClient(&containerGroupsClient.Client)
	if err != nil {
		return storage.FileSharesClient{}, err
	}
	containerGroupsClient.PollingDelay = 5 * time.Second
	containerGroupsClient.RetryAttempts = 30
	containerGroupsClient.RetryDuration = 1 * time.Second
	return containerGroupsClient, nil
}

// NewSubscriptionsClient get subscription client
func NewSubscriptionsClient() (subscription.SubscriptionsClient, error) {
	subc := subscription.NewSubscriptionsClient()
	err := setupClient(&subc.Client)
	if err != nil {
		return subscription.SubscriptionsClient{}, errors.Wrap(errdefs.ErrLoginRequired, err.Error())
	}
	return subc, nil
}

// NewGroupsClient get client to manipulate groups
func NewGroupsClient(subscriptionID string) (resources.GroupsClient, error) {
	groupsClient := resources.NewGroupsClient(subscriptionID)
	err := setupClient(&groupsClient.Client)
	if err != nil {
		return resources.GroupsClient{}, err
	}
	return groupsClient, nil
}

// NewContainerClient get client to manipulate containers
func NewContainerClient(subscriptionID string) (containerinstance.ContainersClient, error) {
	containerClient := containerinstance.NewContainersClient(subscriptionID)
	err := setupClient(&containerClient.Client)
	if err != nil {
		return containerinstance.ContainersClient{}, err
	}
	return containerClient, nil
}
