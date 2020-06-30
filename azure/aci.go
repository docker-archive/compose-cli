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

package azure

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/containerinstance/mgmt/2018-10-01/containerinstance"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	tm "github.com/buger/goterm"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/pkg/errors"

	"github.com/docker/api/azure/login"
	"github.com/docker/api/context/store"
	"github.com/docker/api/progress"
)

const aciDockerUserAgent = "docker-cli"

func createACIContainers(ctx context.Context, aciContext store.AciContext, groupDefinition containerinstance.ContainerGroup) error {
	containerGroupsClient, err := getContainerGroupsClient(aciContext.SubscriptionID)
	if err != nil {
		return errors.Wrapf(err, "cannot get container group client")
	}

	// Check if the container group already exists
	_, err = containerGroupsClient.Get(ctx, aciContext.ResourceGroup, *groupDefinition.Name)
	if err != nil {
		if err, ok := err.(autorest.DetailedError); ok {
			if err.StatusCode != http.StatusNotFound {
				return err
			}
		} else {
			return err
		}
	} else {
		return fmt.Errorf("container group %q already exists", *groupDefinition.Name)
	}

	return createOrUpdateACIContainers(ctx, aciContext, groupDefinition)
}

func createOrUpdateACIContainers(ctx context.Context, aciContext store.AciContext, groupDefinition containerinstance.ContainerGroup) error {
	w := progress.ContextWriter(ctx)
	containerGroupsClient, err := getContainerGroupsClient(aciContext.SubscriptionID)
	if err != nil {
		return errors.Wrapf(err, "cannot get container group client")
	}
	w.Event(progress.Event{
		ID:         *groupDefinition.Name,
		Status:     progress.Working,
		StatusText: "Waiting",
	})

	future, err := containerGroupsClient.CreateOrUpdate(
		ctx,
		aciContext.ResourceGroup,
		*groupDefinition.Name,
		groupDefinition,
	)
	if err != nil {
		return err
	}

	w.Event(progress.Event{
		ID:         *groupDefinition.Name,
		Status:     progress.Done,
		StatusText: "Created",
	})
	for _, c := range *groupDefinition.Containers {
		w.Event(progress.Event{
			ID:         *c.Name,
			Status:     progress.Working,
			StatusText: "Waiting",
		})
	}

	err = future.WaitForCompletionRef(ctx, containerGroupsClient.Client)
	if err != nil {
		return err
	}

	for _, c := range *groupDefinition.Containers {
		w.Event(progress.Event{
			ID:         *c.Name,
			Status:     progress.Done,
			StatusText: "Done",
		})
	}

	return err
}

func getACIContainerGroup(ctx context.Context, aciContext store.AciContext, containerGroupName string) (containerinstance.ContainerGroup, error) {
	containerGroupsClient, err := getContainerGroupsClient(aciContext.SubscriptionID)
	if err != nil {
		return containerinstance.ContainerGroup{}, fmt.Errorf("cannot get container group client: %v", err)
	}

	return containerGroupsClient.Get(ctx, aciContext.ResourceGroup, containerGroupName)
}

func deleteACIContainerGroup(ctx context.Context, aciContext store.AciContext, containerGroupName string) (containerinstance.ContainerGroup, error) {
	containerGroupsClient, err := getContainerGroupsClient(aciContext.SubscriptionID)
	if err != nil {
		return containerinstance.ContainerGroup{}, fmt.Errorf("cannot get container group client: %v", err)
	}

	return containerGroupsClient.Delete(ctx, aciContext.ResourceGroup, containerGroupName)
}

func execACIContainer(ctx context.Context, aciContext store.AciContext, command, containerGroup string, containerName string) (c containerinstance.ContainerExecResponse, err error) {
	containerClient, err := getContainerClient(aciContext.SubscriptionID)
	if err != nil {
		return c, errors.Wrapf(err, "cannot get container client")
	}
	rows, cols := getTermSize()
	containerExecRequest := containerinstance.ContainerExecRequest{
		Command: to.StringPtr(command),
		TerminalSize: &containerinstance.ContainerExecRequestTerminalSize{
			Rows: rows,
			Cols: cols,
		},
	}

	return containerClient.ExecuteCommand(
		ctx,
		aciContext.ResourceGroup,
		containerGroup,
		containerName,
		containerExecRequest)
}

func getTermSize() (*int32, *int32) {
	rows := tm.Height()
	cols := tm.Width()
	return to.Int32Ptr(int32(rows)), to.Int32Ptr(int32(cols))
}

func exec(ctx context.Context, address string, password string, reader io.Reader, writer io.Writer) error {
	conn, _, _, err := ws.DefaultDialer.Dial(ctx, address)
	if err != nil {
		return err
	}
	err = wsutil.WriteClientMessage(conn, ws.OpText, []byte(password))
	if err != nil {
		return err
	}

	downstreamChannel := make(chan error, 10)
	upstreamChannel := make(chan error, 10)
	done := make(chan struct{})

	var commandBuff string
	go func() {
		defer close(done)
		for {
			msg, _, err := wsutil.ReadServerData(conn)
			msgStr := string(msg)
			if err != nil {
				// fmt.Print("return: ", err)
				if err == io.EOF {
					downstreamChannel <- nil
					return
				}
				downstreamChannel <- err
				return
			}
			// fmt.Print("From WS: ", string(msg), "\n")

			// for i, c := range msg {
			// 	fmt.Printf("%d msg        : %d -> %s\n", i, c, string(msg[i]))
			// }
			// for i, c := range commandBuff {
			// 	fmt.Printf("%d commandBuff: %d -> %s\n", i, c, string(commandBuff[i]))
			// }

			// fmt.Printf("\n(%d, %d)\n(\"commandBuff\", \"msgStr\")\n(\"%s\", \"%s\")\n",
			// 	len(commandBuff), len(msgStr),
			// 	commandBuff, msgStr)

			/*
				if commandBuff != "" && strings.HasPrefix(msgStr, strings.TrimSpace(commandBuff)) {
					// fmt.Println("GOOOOOOOOTTTTTTTTTT IT")
					commandBuff = ""
					msgStr = msgStr[len(commandBuff):]
					continue
				} else {
					// fmt.Println("GOOOOOOOOTTTTTTTTTT AAAAAAT")
				}
			*/

			commandBuff, msgStr = popCommonPrefix(commandBuff, msgStr)
			_, _ = fmt.Fprint(writer, msgStr)
		}
	}()

	go func() {
		for {
			// We send each byte, byte-per-byte over the
			// websocket because the console is in raw mode
			buffer := make([]byte, 1)
			n, err := reader.Read(buffer)
			if err != nil {
				if err == io.EOF {
					upstreamChannel <- nil
					return
				}
				// fmt.Println("return from upstreamChannel errorF", err)
				upstreamChannel <- err
				return
			}

			// fmt.Print(n, " To WS: ", string(buffer), "\n")
			if n > 0 {
				if buffer[0] == '\n' {
					commandBuff += "\r"
				}
				commandBuff = commandBuff + string(buffer)
				// if buffer[0] == '\n' {
				// 	for i, c := range commandBuff {
				// 		fmt.Printf("%d Before commandBuff: %d -> %s\n", i, c, string(commandBuff[i]))
				// 	}
				// 	commandBuff += "\r"
				// 	for i, c := range commandBuff {
				// 		fmt.Printf("%d After  commandBuff: %d -> %s\n", i, c, string(commandBuff[i]))
				// 	}
				// }
				err := wsutil.WriteClientMessage(conn, ws.OpText, buffer)
				if err != nil {
					// fmt.Println("return from upstreamChannel", err)
					upstreamChannel <- err
					return
				}
			}
		}
	}()

	for {
		select {
		case <-done:
			return nil
		case err := <-downstreamChannel:
			if err != nil {
				return errors.Wrap(err, "failed to read input from container")
			}
		case err := <-upstreamChannel:
			if err != nil {
				return errors.Wrap(err, "failed to send input to container")
			}
		}
	}
}

func popCommonPrefix(a, b string) (string, string) {
	for len(a) > 0 && len(b) > 0 &&
		a[0] == b[0] &&
		a[0] != '\n' {
		a = a[1:]
		b = b[1:]
	}
	return a, b
}

func getACIContainerLogs(ctx context.Context, aciContext store.AciContext, containerGroupName, containerName string, tail *int32) (string, error) {
	containerClient, err := getContainerClient(aciContext.SubscriptionID)
	if err != nil {
		return "", errors.Wrapf(err, "cannot get container client")
	}

	logs, err := containerClient.ListLogs(ctx, aciContext.ResourceGroup, containerGroupName, containerName, tail)
	if err != nil {
		return "", fmt.Errorf("cannot get container logs: %v", err)
	}
	return *logs.Content, err
}

func getContainerGroupsClient(subscriptionID string) (containerinstance.ContainerGroupsClient, error) {
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
	aciClient.UserAgent = aciDockerUserAgent
	auth, err := login.NewAuthorizerFromLogin()
	if err != nil {
		return err
	}
	aciClient.Authorizer = auth
	return nil
}

func getContainerClient(subscriptionID string) (containerinstance.ContainerClient, error) {
	containerClient := containerinstance.NewContainerClient(subscriptionID)
	err := setupClient(&containerClient.Client)
	if err != nil {
		return containerinstance.ContainerClient{}, err
	}
	return containerClient, nil
}
