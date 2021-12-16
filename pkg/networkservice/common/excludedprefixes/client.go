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
	"net"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
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

	logger := log.FromContext(ctx).WithField("ExcludedPrefixesClient", "Request")
	ipCtx := conn.GetContext().GetIpContext()

	var newExcludedPrefixes []string
	oldExcludedPrefixes := ipCtx.GetExcludedPrefixes()
	if len(epc.excludedPrefixes) > 0 {
		<-epc.executor.AsyncExec(func() {
			logger.Debugf("Adding new excluded IPs to the request: %+v", epc.excludedPrefixes)
			newExcludedPrefixes = ipCtx.GetExcludedPrefixes()
			newExcludedPrefixes = append(newExcludedPrefixes, epc.excludedPrefixes...)
			newExcludedPrefixes = RemoveDuplicates(newExcludedPrefixes)

			// excluding IPs for current request/connection before calling next client for the refresh use-case
			newExcludedPrefixes = Exclude(newExcludedPrefixes, append(ipCtx.GetSrcIpAddrs(), ipCtx.GetDstIpAddrs()...))

			logger.Debugf("Excluded prefixes from request - %+v", newExcludedPrefixes)
			ipCtx.ExcludedPrefixes = newExcludedPrefixes
		})
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	resp, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		ipCtx.ExcludedPrefixes = oldExcludedPrefixes
		return resp, err
	}

	respIPContext := resp.GetContext().GetIpContext()

	err = validateIPs(respIPContext, newExcludedPrefixes)
	if err != nil {
		closeCtx, cancelFunc := postponeCtxFunc()
		defer cancelFunc()

		logger.Errorf("Source or destination IPs are overlapping with excluded prefixes, srcIPs: %+v, dstIPs: %+v, excluded prefixes: %+v, error: %s",
			respIPContext.GetSrcIpAddrs(), respIPContext.GetDstIpAddrs(), newExcludedPrefixes, err.Error())

		if _, closeErr := next.Client(ctx).Close(closeCtx, conn, opts...); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	logger.Debugf("Request excluded IPs - srcIPs: %v, dstIPs: %v, excluded prefixes: %v", respIPContext.GetSrcIpAddrs(),
		respIPContext.GetDstIpAddrs(), respIPContext.GetExcludedPrefixes())

	<-epc.executor.AsyncExec(func() {
		epc.excludedPrefixes = append(epc.excludedPrefixes, respIPContext.GetSrcIpAddrs()...)
		epc.excludedPrefixes = append(epc.excludedPrefixes, respIPContext.GetDstIpAddrs()...)
		epc.excludedPrefixes = append(epc.excludedPrefixes, getRoutePrefixes(respIPContext.GetSrcRoutes())...)
		epc.excludedPrefixes = append(epc.excludedPrefixes, getRoutePrefixes(respIPContext.GetDstRoutes())...)
		epc.excludedPrefixes = append(epc.excludedPrefixes, respIPContext.GetExcludedPrefixes()...)
		epc.excludedPrefixes = RemoveDuplicates(epc.excludedPrefixes)
		logger.Debugf("Added excluded prefixes: %+v", epc.excludedPrefixes)
	})

	return resp, err
}

func (epc *excludedPrefixesClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("ExcludedPrefixesClient", "Close")
	ipCtx := conn.GetContext().GetIpContext()

	<-epc.executor.AsyncExec(func() {
		epc.excludedPrefixes = Exclude(epc.excludedPrefixes, ipCtx.GetSrcIpAddrs())
		epc.excludedPrefixes = Exclude(epc.excludedPrefixes, ipCtx.GetDstIpAddrs())
		epc.excludedPrefixes = Exclude(epc.excludedPrefixes, getRoutePrefixes(ipCtx.GetSrcRoutes()))
		epc.excludedPrefixes = Exclude(epc.excludedPrefixes, getRoutePrefixes(ipCtx.GetDstRoutes()))
		epc.excludedPrefixes = Exclude(epc.excludedPrefixes, ipCtx.GetExcludedPrefixes())
		logger.Debugf("Excluded prefixes after closing connection: %+v", epc.excludedPrefixes)
	})

	return next.Client(ctx).Close(ctx, conn, opts...)
}

func getRoutePrefixes(routes []*networkservice.Route) []string {
	var rv []string
	for _, route := range routes {
		rv = append(rv, route.GetPrefix())
	}

	return rv
}

func validateIPs(ipContext *networkservice.IPContext, excludedPrefixes []string) error {
	ip4Pool := ippool.New(net.IPv4len)
	ip6Pool := ippool.New(net.IPv6len)

	for _, prefix := range excludedPrefixes {
		_, ipNet, err := net.ParseCIDR(prefix)
		if err != nil {
			return err
		}

		ip4Pool.AddNet(ipNet)
		ip6Pool.AddNet(ipNet)
	}

	prefixes := make([]string, 0, len(ipContext.GetSrcIpAddrs())+len(ipContext.GetDstIpAddrs()))
	prefixes = append(prefixes, ipContext.GetSrcIpAddrs()...)
	prefixes = append(prefixes, ipContext.GetDstIpAddrs()...)

	for _, prefix := range prefixes {
		ip, _, err := net.ParseCIDR(prefix)
		if err != nil {
			return err
		}

		if ip4Pool.Contains(ip) || ip6Pool.Contains(ip) {
			return errors.Errorf("IP %v is excluded, but it was found in response IPs", ip)
		}
	}

	return nil
}
