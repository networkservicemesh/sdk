// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package excludedprefixes

import (
	"context"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type excludedPrefixesClient struct {
	excludedPrefixes []string
	executor         serialize.Executor
}

// NewClient - creates a networkservice.NetworkServiceClient chain element that excludes prefixes already used by other NetworkServices
func NewClient() networkservice.NetworkServiceClient {
	return &excludedPrefixesClient{
		excludedPrefixes: make([]string, 0),
	}
}

func (epc *excludedPrefixesClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn := request.GetConnection()
	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}

	if conn.GetContext().GetIpContext() == nil {
		conn.Context.IpContext = &networkservice.IPContext{}
	}

	ipCtx := conn.GetContext().GetIpContext()

	if len(epc.excludedPrefixes) > 0 {
		<-epc.executor.AsyncExec(func() {
			log.FromContext(ctx).Debugf("ExcludedPrefixesClient: adding new excluded IPs to the request: %v", epc.excludedPrefixes)
			excludedPrefixes := ipCtx.GetExcludedPrefixes()
			excludedPrefixes = append(excludedPrefixes, epc.excludedPrefixes...)
			excludedPrefixes = removeDuplicates(excludedPrefixes)

			log.FromContext(ctx).Debugf("ExcludedPrefixesClient: excluded prefixes from request - %v", excludedPrefixes)
			ipCtx.ExcludedPrefixes = excludedPrefixes
		})
	}

	resp, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return resp, err
	}

	log.FromContext(ctx).Debugf("ExcludedPrefixesClient: request excluded IPs - srcIPs: %v, dstIPs: %v",
		resp.GetContext().GetIpContext().GetSrcIpAddrs(), resp.GetContext().GetIpContext().GetDstIpAddrs())

	<-epc.executor.AsyncExec(func() {
		epc.excludedPrefixes = append(epc.excludedPrefixes, ipCtx.GetSrcIpAddrs()...)
		epc.excludedPrefixes = append(epc.excludedPrefixes, ipCtx.GetDstIpAddrs()...)
		epc.excludedPrefixes = append(epc.excludedPrefixes, ipCtx.GetExcludedPrefixes()...)
		epc.excludedPrefixes = removeDuplicates(epc.excludedPrefixes)
		log.FromContext(ctx).Debugf("Added excluded prefixes: %v", epc.excludedPrefixes)
	})

	return resp, err
}

func (epc *excludedPrefixesClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	ipCtx := conn.GetContext().GetIpContext()

	<-epc.executor.AsyncExec(func() {
		epc.excludedPrefixes = exclude(epc.excludedPrefixes, ipCtx.GetSrcIpAddrs())
		epc.excludedPrefixes = exclude(epc.excludedPrefixes, ipCtx.GetDstIpAddrs())
		epc.excludedPrefixes = exclude(epc.excludedPrefixes, ipCtx.GetExcludedPrefixes())
		log.FromContext(ctx).Debugf("Excluded prefixes after closing connection: %v", epc.excludedPrefixes)
	})

	return next.Client(ctx).Close(ctx, conn, opts...)
}
