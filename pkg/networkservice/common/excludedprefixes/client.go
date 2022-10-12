// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type awarenessGroup struct {
	NSUrlSet          map[url.URL]struct{}
	ExcludedPrfixes   []string
	ConnectionCounter int32
}

func (g *awarenessGroup) contains(nsURL *url.URL) bool {
	_, ok := g.NSUrlSet[*nsURL]
	return ok
}

type excludedPrefixesClient struct {
	excludedPrefixes []string
	awarenessGroups  []*awarenessGroup
	executor         serialize.Executor
}

// NewClient - creates a networkservice.NetworkServiceClient chain element that excludes prefixes already used by other NetworkServices
func NewClient(opts ...ClientOption) networkservice.NetworkServiceClient {
	client := &excludedPrefixesClient{
		excludedPrefixes: make([]string, 0),
		awarenessGroups:  make([]*awarenessGroup, 0),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

func (epc *excludedPrefixesClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn := request.GetConnection()
	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}

	if conn.GetContext().GetIpContext() == nil {
		conn.Context.IpContext = &networkservice.IPContext{}
	}

	nsurl := getNSURL(request)
	groupIndex := checkAwarenessGroups(nsurl, epc.awarenessGroups)

	var awarenessGroupsExcludedPrefixes []string
	for i, group := range epc.awarenessGroups {
		if i != groupIndex {
			awarenessGroupsExcludedPrefixes = append(awarenessGroupsExcludedPrefixes, group.ExcludedPrfixes...)
		}
	}

	logger := log.FromContext(ctx).WithField("ExcludedPrefixesClient", "Request")
	ipCtx := conn.GetContext().GetIpContext()

	var newExcludedPrefixes []string
	oldExcludedPrefixes := ipCtx.GetExcludedPrefixes()
	if len(epc.excludedPrefixes) > 0 || len(awarenessGroupsExcludedPrefixes) > 0 {
		<-epc.executor.AsyncExec(func() {
			logger.Debugf("Adding new excluded IPs to the request: %+v", epc.excludedPrefixes)
			newExcludedPrefixes = ipCtx.GetExcludedPrefixes()
			newExcludedPrefixes = append(newExcludedPrefixes, epc.excludedPrefixes...)
			newExcludedPrefixes = append(newExcludedPrefixes, awarenessGroupsExcludedPrefixes...)
			newExcludedPrefixes = removeDuplicates(newExcludedPrefixes)

			// excluding IPs for current request/connection before calling next client for the refresh use-case
			newExcludedPrefixes = exclude(newExcludedPrefixes, append(ipCtx.GetSrcIpAddrs(), ipCtx.GetDstIpAddrs()...))

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
		var excludedPrefixes []string
		excludedPrefixes = append(excludedPrefixes, respIPContext.GetSrcIpAddrs()...)
		excludedPrefixes = append(excludedPrefixes, respIPContext.GetDstIpAddrs()...)
		excludedPrefixes = append(excludedPrefixes, getRoutePrefixes(respIPContext.GetSrcRoutes())...)
		excludedPrefixes = append(excludedPrefixes, getRoutePrefixes(respIPContext.GetDstRoutes())...)

		if groupIndex >= 0 {
			epc.awarenessGroups[groupIndex].ExcludedPrfixes = append(epc.awarenessGroups[groupIndex].ExcludedPrfixes, excludedPrefixes...)
			epc.awarenessGroups[groupIndex].ExcludedPrfixes = removeDuplicates(epc.awarenessGroups[groupIndex].ExcludedPrfixes)
			atomic.AddInt32(&epc.awarenessGroups[groupIndex].ConnectionCounter, 1)
		} else {
			excludedPrefixes = append(excludedPrefixes, respIPContext.GetExcludedPrefixes()...)
			epc.excludedPrefixes = append(epc.excludedPrefixes, excludedPrefixes...)
			epc.excludedPrefixes = removeDuplicates(epc.excludedPrefixes)
		}

		logger.Debugf("Added excluded prefixes: %+v", epc.excludedPrefixes)
	})

	ipCtx.ExcludedPrefixes = oldExcludedPrefixes

	return resp, err
}

func (epc *excludedPrefixesClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("ExcludedPrefixesClient", "Close")
	ipCtx := conn.GetContext().GetIpContext()

	<-epc.executor.AsyncExec(func() {
		epc.excludedPrefixes = exclude(epc.excludedPrefixes, ipCtx.GetSrcIpAddrs())
		epc.excludedPrefixes = exclude(epc.excludedPrefixes, ipCtx.GetDstIpAddrs())
		epc.excludedPrefixes = exclude(epc.excludedPrefixes, getRoutePrefixes(ipCtx.GetSrcRoutes()))
		epc.excludedPrefixes = exclude(epc.excludedPrefixes, getRoutePrefixes(ipCtx.GetDstRoutes()))
		epc.excludedPrefixes = exclude(epc.excludedPrefixes, ipCtx.GetExcludedPrefixes())

		nsurl := getNSURL(&networkservice.NetworkServiceRequest{Connection: conn})
		groupIndex := checkAwarenessGroups(nsurl, epc.awarenessGroups)
		if groupIndex >= 0 {
			atomic.AddInt32(&epc.awarenessGroups[groupIndex].ConnectionCounter, -1)
			if atomic.LoadInt32(&epc.awarenessGroups[groupIndex].ConnectionCounter) == 0 {
				epc.awarenessGroups[groupIndex].ExcludedPrfixes = make([]string, 0)
			}
		}

		logger.Debugf("Excluded prefixes after closing connection: %+v", epc.excludedPrefixes)
	})

	return next.Client(ctx).Close(ctx, conn, opts...)
}

func getRoutePrefixes(routes []*networkservice.Route) []string {
	var rv []string
	for _, route := range routes {
		var o, b = route.GetPrefixIPNet().Mask.Size()
		if o != b {
			continue
		}
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
		// TODO: Think about validating routes with prefixes size less than /32 and /128
		if prefixLen, maxLen := ipNet.Mask.Size(); prefixLen != maxLen {
			continue
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

func getNSURL(request *networkservice.NetworkServiceRequest) *url.URL {
	nsurl := &url.URL{}

	nsurl.Host = request.GetConnection().GetNetworkService()
	mechanism := request.GetConnection().GetMechanism()
	if mechanism == nil && len(request.MechanismPreferences) > 0 {
		mechanism = request.MechanismPreferences[0]
	}

	nsurl.Scheme = strings.ToLower(mechanism.GetType())
	iface := mechanism.GetParameters()[common.InterfaceNameKey]
	if iface != "" {
		nsurl.Path = "/" + iface
	}
	query := nsurl.Query()
	for k, v := range request.GetConnection().GetLabels() {
		query.Add(k, v)
	}
	nsurl.RawQuery = query.Encode()

	return nsurl
}

func checkAwarenessGroups(nsurl *url.URL, awarenessGroups []*awarenessGroup) int {
	for i, group := range awarenessGroups {
		if group.contains(nsurl) {
			return i
		}
	}

	return -1
}
