// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

// Package swapip provides chain element to swapping fields of remote mechanisms such as common.SrcIP and common.DstIP
// from internal to external and vice versa on response.
package swapip

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type swapIPClient struct {
	internalToExternalMap *atomic.Value
}

// NewClient creates new swap chain element. Expects public IP address of node.
func NewClient(updateIPMapCh <-chan map[string]string) networkservice.NetworkServiceClient {
	v := new(atomic.Value)
	v.Store(map[string]string{})
	go func() {
		for data := range updateIPMapCh {
			v.Store(data)
		}
	}()
	return &swapIPClient{internalToExternalMap: v}
}

func (i *swapIPClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	internalToExternalMap := i.internalToExternalMap.Load().(map[string]string)
	mechanisms := request.GetMechanismPreferences()
	var isSourceSide bool

	if m := request.GetConnection().GetMechanism(); m != nil {
		mechanisms = append(mechanisms, m)
	}

	for _, m := range mechanisms {
		params := m.GetParameters()
		if params != nil {
			srcIP, ok := params[common.SrcIP]
			if !ok {
				continue
			}
			isSourceSide = params[common.SrcOriginalIP] == ""
			if isSourceSide {
				params[common.SrcIP], params[common.SrcOriginalIP] = internalToExternalMap[srcIP], srcIP
			}
		}
	}

	resp, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	params := resp.GetMechanism().GetParameters()
	if params != nil {
		if isSourceSide {
			params[common.SrcIP], params[common.SrcOriginalIP] = params[common.SrcOriginalIP], ""
		}
	}

	return resp, nil
}

func (i *swapIPClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
