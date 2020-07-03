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

package errdefs

import (
	"github.com/pkg/errors"
)

const (
	//ExitCodeLoginRequired exit code when command cannot execute because it requires cloud login
	// This will be used by VSCode to detect when creating context if the user needs to login first
	ExitCodeLoginRequired = 5
)

// Generic errors
var (
	// ErrAlreadyExists is returned when an object already exists
	ErrAlreadyExists = errors.New("already exists")
	// ErrForbidden is returned when an operation is not permitted
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound is returned when an object is not found
	ErrNotFound = errors.New("not found")
	// ErrNotImplemented is returned when a backend doesn't implement
	// an action
	ErrNotImplemented = errors.New("not implemented")
	// ErrParsingFailed is returned when a string cannot be parsed
	ErrParsingFailed = errors.New("parsing failed")

	// ErrUnknown is returned when the cause of the error is not known or
	// handled
	ErrUnknown = errors.New("unknown")
)

// Specific errors
var (
	// ErrImageInaccessible is returned when an image does not exist or there
	// are permissions issues fetching it
	ErrImageInaccessible = errors.New("image inaccessible")
	// ErrInvalidName is returned when a object is given an invalid name
	ErrInvalidName = errors.New("invalid name")
	// ErrLoginFailed is returned when login failed
	ErrLoginFailed = errors.New("login failed")
	// ErrLoginRequired is returned when login is required for a specific action
	ErrLoginRequired = errors.New("login required")
	// ErrPortMappingUnsupported is returned when a backend does not support
	// port mapping
	ErrPortMappingUnsupported = errors.New("port mapping unsupported")
	// ErrWrongContextType is returned when the caller tries to get a context
	// with the wrong type
	ErrWrongContextType = errors.New("wrong context type")
)
