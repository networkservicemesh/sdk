// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type swapIPServer struct {
	internalToExternalMap *atomic.Value
}

func (i *swapIPServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	internalToExternalMap := i.internalToExternalMap.Load().(map[string]string)
	mechanisms := request.GetMechanismPreferences()
	var isSourceSide bool

	if m := request.GetConnection().GetMechanism(); m != nil {
		mechanisms = append(mechanisms, m)
	}

	for _, m := range mechanisms {
		params := m.GetParameters()
		if params != nil {
			_, ok := params[common.SrcIP]
			if !ok {
				continue
			}
			isSourceSide = params[common.SrcOriginalIP] == ""
			if !isSourceSide {
				params[common.DstOriginalIP] = ""
				params[common.DstIP] = ""
			}
		}
	}

	resp, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	params := resp.GetMechanism().GetParameters()

	if params != nil {
		if !isSourceSide {
			dstIP := params[common.DstIP]
			params[common.DstIP], params[common.DstOriginalIP] = internalToExternalMap[dstIP], dstIP
		}
	}

	return resp, nil
}

func (i *swapIPServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

// NewServer creates new swap chain element. Expects public IP address of node.
func NewServer(updateIPMapCh <-chan map[string]string) networkservice.NetworkServiceServer {
	v := new(atomic.Value)
	v.Store(map[string]string{})
	go func() {
		for data := range updateIPMapCh {
			v.Store(data)
		}
	}()
	return &swapIPServer{internalToExternalMap: v}
}
