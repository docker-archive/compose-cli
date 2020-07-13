# Errors

## Context

The quality of the existing CLI errors is mixed.
In the first example, the user tries to remove an image that is not in their
local image store.
The error is clear and concise.

```console
$ docker rmi hello
Error: No such image: hello
```

In this next example, the user tries to run a container using an image that they
do not have access to.
This is a complex case as the image pull is implicit and it is not possible to
differentiate between the user not having access to an existing image or if the
image exists at all.

```console
$ docker run hello
Unable to find image 'hello:latest' locally
docker: Error response from daemon: pull access denied for hello, repository does not exist or may require 'docker login': denied: requested access to the resource is denied.
See 'docker run --help'.
```

The second example leaks internal implementation details (like that the image
store is part of the daemon) and provides only one of the two possible fixes:
It proposes logging into the registry when the issue could be that the user
specified the wrong image.

We should see our errors as an opportunity to inform and educate users about
our platform.
The goal should be that a user does not need to leave their terminal (or tool
that consumes our platform) to debug common issues.

## Design Considerations

There are two types of consumers of the CLI and the API:
1. End users
2. Developers building on top of the CLI and API

Most end users will consume the CLI and will not be familiar with Docker
internals.
As such, we should describe processes taking place and give them
actionable error messages.
This will help them to learn along the way and allow them to unblock themselves
if they get stuck.

Developers building on top of the API and CLI will likely have a deeper Docker
knowledge.
They will want errors to accurately describe what went wrong and will
want them to be typed so that they can be handled.
For processes that take time, developers will want to surface updates to their
users.

This means we need two mechanisms:
1. A system of typed errors
2. A system of typed events

This document focuses on errors and leaves the event system as future work.

The API currently maps 1:1 with the CLI commands which means that the
information returned as part of errors is common.
The difference between API and CLI errors is how they are rendered to the user:
The API errors need to be typed so that developers can handle them while the CLI
errors need to be written to the terminal clearly so that the user can easily
read them.

## General Requirements

- The CLI should be treated as a consumer of the API.
- All errors and events must pass through interface; no backend may write
  directly to the user.
- The errors and events must work over gRPC and with languages other than Go.
- Outputs should be available for TTY, plain (i.e.: pure text), and JSON.

## Implementation

The errors are defined in the API's `errdefs` package which leverages the
popular `github.com/pkg/errors` library for error handling.

The following shows an example of how errors can be handled.
Note that the functions are ordered so that as you read the code, you descend
the stack.

```golang
package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
)

// Generic errors that are used by low level functions and packages
var (
	// ErrNetwork returned when there is a networking error
	ErrNetwork = errors.New("network error")
	// ErrNotFound returned when something cannot be found
	ErrNotFound = errors.New("not found")
)

// Typed errors that are used higher up the stack and that can be used to give
// the user more specific help
var (
	// ErrImageInaccessible is returned when an image is not found or if the
	// user does not have permission to access it
	ErrImageInaccessible = errors.New("image inaccessible")
)

// BackendInterface shared across backends
type BackendInterface interface {
	Run() error
}

// Backend implements BackendInterface
type Backend struct{}

func main() {
	b := Backend{}
	if err := b.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		if errors.Is(err, ErrImageInaccessible) {
			fmt.Println("fix: check the image name or ensure that you're logged into the registry")
		}
		if errors.Is(err, ErrNetwork) {
			fmt.Println("fix: check your network connection")
		}
		os.Exit(1)
	}
}

// Run starts a container
func (b Backend) Run() error {
	if err := CreateContainer(); err != nil {
		// Wrap the error with context that we have here
		return errors.Wrapf(err, "unable to run container")
	}
	return nil
}

// CreateContainer does the container create part of run
func CreateContainer() error {
	err := FetchImage()
	if errors.Is(err, ErrNotFound) {
		// In the case that we get a "not found" error, we translate it into a
		// more specific user so that we can use it higher up the stack
		return ErrImageInaccessible
	}
	// Return all other errors up the stack
	return err
}

// FetchImage fetches the container image from the store
func FetchImage() error {
	// Either return the raw error or a generic one from low down in the stack
	return ErrNotFound
	// return ErrNetwork
}
```

Where to change error types and where to wrap errors with more context is a
function of what the end user needs to know and the code structure.
It is expected that this will be iterative work as we discover which errors
users commonly run into and so we need to specially handle.

### CLI Errors

To make CLI errors easier to distinguish in the terminal, we should make use of
colors where they are supported (i.e.: when there is a TTY).
For common issues, we should also add fix messages that point users in the right
direction.

## Events

The current progress writer could act as a basis for an events system.
It has the ability to report the status of a currently running process along
with a message.
This is sufficient for providing CLI feedback to users but it would need to be
extended to be session based for the API.
