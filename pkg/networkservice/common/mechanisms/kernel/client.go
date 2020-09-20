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

package kernel

import (
	"context"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/fs"
)

type mechanismsClient struct {
	ifName string
}

// NewClient - creates a NetworkServiceClient that that requests a kernel interface and populates the netns inode
func NewClient(ifName string) networkservice.NetworkServiceClient {
	return &mechanismsClient{
		ifName: ifName,
	}
}

func (c *mechanismsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.GetConnection().GetMechanism() == nil {
		netNSInode, err := fs.GetInode(netNSFilename)
		if err != nil {
			return nil, err
		}
		request.MechanismPreferences = append(request.MechanismPreferences, &networkservice.Mechanism{
			Cls:  cls.LOCAL,
			Type: kernel.MECHANISM,
			Parameters: map[string]string{
				kernel.NetNSInodeKey:    strconv.FormatUint(uint64(netNSInode), 10),
				kernel.InterfaceNameKey: c.ifName,
				kernel.NetNSURL:         netNSUrl,
			},
		})
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *mechanismsClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
