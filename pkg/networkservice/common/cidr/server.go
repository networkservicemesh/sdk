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

package cidr

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/prefixpool"
)

type cidrServer struct {
	pool *prefixpool.PrefixPool

	prefixes        []string
	excludePrefixes []string
	once            sync.Once
	initErr         error
}

// NewServer creates a NetworkServiceServer chain element that allocates CIDR from some global prefix
// and saves it in ExtraPrefix
func NewServer(prefixes, excludePrefixes []string) networkservice.NetworkServiceServer {
	return &cidrServer{
		prefixes:        prefixes,
		excludePrefixes: excludePrefixes,
	}
}

func (c *cidrServer) init() {
	if len(c.prefixes) == 0 {
		c.initErr = errors.New("required one or more prefixes")
		return
	}
	var err error
	if c.pool, err = prefixpool.New(c.prefixes...); err != nil {
		c.initErr = errors.New("required one or more prefixes")
		return
	}
	if _, err = c.pool.ExcludePrefixes(c.excludePrefixes); err != nil {
		c.initErr = errors.New("unable to exclude additional prefixes")
		return
	}
}

func (c *cidrServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	c.once.Do(c.init)
	if c.initErr != nil {
		return nil, c.initErr
	}

	conn := request.GetConnection()
	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}
	if conn.GetContext().GetIpContext() == nil {
		conn.Context.IpContext = &networkservice.IPContext{}
	}
	ipContext := conn.GetContext().GetIpContext()

	/* If we already have extra prefixes, exclude them from the next CIDR allocations */
	if ipContext.GetExtraPrefixes() != nil {
		if _, err := c.pool.ExcludePrefixes(ipContext.GetExtraPrefixes()); err != nil {
			return nil, err
		}
	} else if ipContext.GetExtraPrefixRequest() != nil {
		/* Else, extract a new one if there is ExtraPrefixRequest */
		requested, err := c.pool.ExtractPrefixes(request.GetConnection().GetId(), ipContext.GetExtraPrefixRequest()...)
		if err != nil {
			return nil, err
		}
		ipContext.ExtraPrefixes = append(ipContext.ExtraPrefixes, requested...)
		ipContext.ExtraPrefixRequest = nil

		for i := 0; i < len(requested); i++ {
			ipContext.DstRoutes = append(ipContext.DstRoutes, &networkservice.Route{
				Prefix: requested[i],
			})
		}
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		extraPrefixes := request.GetConnection().GetContext().GetIpContext().GetExtraPrefixes()
		if len(extraPrefixes) != 0 {
			_ = c.pool.ReleaseExcludedPrefixes(extraPrefixes)
		}
		return conn, err
	}

	ipContext = conn.GetContext().GetIpContext()
	if ok := load(ctx, metadata.IsClient(c)); !ok {
		/* Set srcRoutes = prefixes because these hosts should be available through this element */
		for i := 0; i < len(c.prefixes); i++ {
			ipContext.SrcRoutes = append(ipContext.SrcRoutes, &networkservice.Route{
				Prefix: c.prefixes[i],
			})
		}
		store(ctx, metadata.IsClient(c))
	}

	return conn, err
}

func (c *cidrServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	extraPrefixes := conn.GetContext().GetIpContext().GetExtraPrefixes()
	if len(extraPrefixes) != 0 {
		_ = c.pool.ReleaseExcludedPrefixes(extraPrefixes)
	}
	delete(ctx, metadata.IsClient(c))

	return next.Server(ctx).Close(ctx, conn)
}
