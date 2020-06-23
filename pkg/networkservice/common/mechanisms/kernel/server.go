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
	"strconv"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	mkernel "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/fs"
)

type mechanismsServer struct{}

func (m mechanismsServer) Request(ctx context.Context, req *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn := req.Connection
	inode := uintptr(0)
	inode, err := fs.GetInode("/proc/self/ns/net")
	if err != nil {
		return nil, err
	}
	if conn.GetMechanism() == nil {
		conn.Mechanism = &networkservice.Mechanism{
			Cls:        cls.LOCAL,
			Type:       mkernel.MECHANISM,
			Parameters: make(map[string]string),
		}
	}
	if conn.GetMechanism().GetParameters() == nil {
		conn.GetMechanism().Parameters = make(map[string]string)
	}

	conn.Mechanism.Parameters[common.NetNSInodeKey] = strconv.FormatUint(uint64(inode), 10)
	return next.Server(ctx).Request(ctx, req)
}

func (m mechanismsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

// NewServer - creates a NetworkServiceServer that requests a kernel interface and populates the netns inode
func NewServer() networkservice.NetworkServiceServer {
	return mechanismsServer{}
}
