// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2023 Cisco Systems, Inc.
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

package authorize

import (
	"context"
	"sync/atomic"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type authorizeClient struct {
	policies   policiesList
	serverPeer atomic.Value
}

// NewClient - returns a new authorization networkservicemesh.NetworkServiceClient
// Authorize client checks rigiht side of path.
func NewClient(opts ...Option) networkservice.NetworkServiceClient {
	o := &options{
		policyPaths: []string{
			"etc/nsm/opa/common/.*.rego",
			"etc/nsm/opa/client/.*.rego",
		},
	}
	for _, opt := range opts {
		opt(o)
	}

	policies, err := opa.PoliciesByFileMask(o.policyPaths...)
	if err != nil {
		panic(errors.Wrap(err, "failed to read policies in NetworkService authorize client").Error())
	}
	var policyList policiesList
	for _, p := range policies {
		policyList = append(policyList, p)
	}

	result := &authorizeClient{
		policies: policyList,
	}
	return result
}

func (a *authorizeClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	var p peer.Peer
	opts = append(opts, grpc.Peer(&p))

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if p != (peer.Peer{}) {
		a.serverPeer.Store(&p)
		ctx = peer.NewContext(ctx, &p)
	}

	if err = a.policies.check(ctx, conn.GetPath()); err != nil {
		if !load(ctx, metadata.IsClient(a)) {
			closeCtx, cancelClose := postponeCtxFunc()
			defer cancelClose()

			if _, closeErr := next.Client(ctx).Close(closeCtx, conn, opts...); closeErr != nil {
				err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
			}
		}
		return nil, err
	}

	store(ctx, metadata.IsClient(a))
	return conn, nil
}

func (a *authorizeClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	p, ok := a.serverPeer.Load().(*peer.Peer)
	if ok && p != nil {
		ctx = peer.NewContext(ctx, p)
	}
	del(ctx, metadata.IsClient(a))

	if err := a.policies.check(ctx, conn.GetPath()); err != nil {
		return nil, err
	}

	return next.Client(ctx).Close(ctx, conn, opts...)
}
