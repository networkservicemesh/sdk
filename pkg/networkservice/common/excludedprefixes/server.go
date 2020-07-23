// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
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

// Package excludedprefixes provides a networkservice.NetworkServiceServer chain element that can read excluded prefixes
// from config map and add them to request to avoid repeated usage.
package excludedprefixes

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/prefixpool"
)

type excludedPrefixesServer struct {
	prefixes   *prefixpool.PrefixPool
	configPath string
}

// Note: request.Connection and Connection.Context should not be nil
func (eps *excludedPrefixesServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logger := trace.Log(ctx)

	conn := request.GetConnection()
	if conn.GetContext().GetIpContext() == nil {
		conn.Context.IpContext = &networkservice.IPContext{}
	}
	prefixes := eps.prefixes.GetPrefixes()
	logger.Infof("ExcludedPrefixesService: adding excluded prefixes to connection: %v", prefixes)
	ipCtx := conn.GetContext().GetIpContext()
	ipCtx.ExcludedPrefixes = removeDuplicates(append(ipCtx.GetExcludedPrefixes(), prefixes...))

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if err = eps.validateConnection(conn); err != nil {
		logger.Errorf("ExcludedPrefixesService: connection is invalid: %v", err)
		_, err = next.Server(ctx).Close(ctx, conn)
		logger.Errorf("ExcludedPrefixesService: Close: %v", err)
		return nil, err
	}

	return conn, nil
}

func (eps *excludedPrefixesServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, connection)
}

// NewServer -  creates a networkservice.NetworkServiceServer chain element that can read excluded prefixes from config
// map and add them to request to avoid repeated usage.
// Note: request.Connection and Connection.Context should not be nil when calling Request
func NewServer(setters ...ServerOption) networkservice.NetworkServiceServer {
	server := &excludedPrefixesServer{
		configPath: prefixpool.PrefixesFilePathDefault,
	}
	for _, setter := range setters {
		setter(server)
	}
	server.prefixes = &prefixpool.NewPrefixPoolReader(server.configPath).PrefixPool
	return server
}

func (eps *excludedPrefixesServer) validateConnection(conn *networkservice.Connection) error {
	if err := conn.IsComplete(); err != nil {
		return err
	}

	ipCtx := conn.GetContext().GetIpContext()
	if err := eps.validateIPAddress(ipCtx.GetSrcIpAddr(), "srcIP"); err != nil {
		return err
	}

	return eps.validateIPAddress(ipCtx.GetDstIpAddr(), "dstIP")
}

func (eps *excludedPrefixesServer) validateIPAddress(ip, ipName string) error {
	if ip == "" {
		return nil
	}
	intersect, err := eps.prefixes.Intersect(ip)
	if err != nil {
		return err
	}
	if intersect {
		return errors.Errorf("%s '%s' intersects excluded prefixes list %v", ipName, ip, eps.prefixes.GetPrefixes())
	}
	return nil
}
