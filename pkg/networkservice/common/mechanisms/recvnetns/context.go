// Copyright (c) 2020 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !windows

package recvnetns

import (
	"context"
)

const (
	netNSFileNamePtr contextKeyType = "netNSFileNamePtr"
	netNSFileName    contextKeyType = "netNSFileName"
	// UnixURLScheme - scheme for unix urls
	UnixURLScheme = "unix"
)

type contextKeyType string

// WithFilenamePtr - store a *string in the context to be populated by the recvnetns.Client
func WithFilenamePtr(parent context.Context, filenamePtr *string) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, netNSFileNamePtr, filenamePtr)
}

// FilenamePtr - retrieve the *string stored by WithFilenamePtr
func FilenamePtr(ctx context.Context) *string {
	if rv, ok := ctx.Value(netNSFileNamePtr).(*string); ok {
		return rv
	}
	return nil
}

// WithFilename - store the filename of the netns in the context
func WithFilename(parent context.Context, filename string) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, netNSFileName, filename)
}

// Filename - retrieve the filename of the netns from the context
func Filename(ctx context.Context) string {
	if rv, ok := ctx.Value(netNSFileName).(string); ok {
		return rv
	}
	return ""
}
