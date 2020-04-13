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
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/sdk/pkg/tools/fs"
	"strconv"
)

type mechanismsServer struct{}

func (m mechanismsServer) Request(ctx context.Context, req *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn := req.Connection
	inode := uint64(0)
	inode, err := fs.GetInode("/proc/self/net/ns")
	if err != nil {
		return nil, err
	}
	conn.Mechanism.Parameters[common.NetNSInodeKey] = strconv.FormatUint(inode, 10)
	return conn, nil
}

func (m mechanismsServer) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	panic("implement me")
}

// NewServer - creates a NetworkServiceServer that requests a kernel interface and populates the netns inode
func NewServer() networkservice.NetworkServiceServer {
	return mechanismsServer{}
}
