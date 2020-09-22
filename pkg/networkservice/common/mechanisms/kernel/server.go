// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package kernel provides the necessary mechanisms to request and inject a kernel interface.
package kernel

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type kernelMechanismServer struct{}

// NewServer - creates a NetworkServiceServer that requests a kernel interface and populates the netns inode
func NewServer() networkservice.NetworkServiceServer {
	return &kernelMechanismServer{}
}

func (m *kernelMechanismServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if mechanism := kernel.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		mechanism.SetNetNSURL((&url.URL{Scheme: "file", Path: netNSFilename}).String())
	}
	return next.Server(ctx).Request(ctx, request)
}

func (m *kernelMechanismServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
