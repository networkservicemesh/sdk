// Copyright (c) 2021 Cisco and/or its affiliates.
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

// Package dial will dial up a grpc.ClientConnInterface if a client *url.URL is provided in the ctx, retrievable by
// clienturlctx.ClientURL(ctx) and put the resulting grpc.ClientConnInterface into the ctx using clientconn.Store(..)
// where it can be retrieved by other chain elements using clientconn.Load(...)
package dial

import (
	"context"
	"io"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

type dialClient struct {
	chainCtx    context.Context
	dialOptions []grpc.DialOption
	dialTimeout time.Duration
}

// NewClient - returns new dial chain element
func NewClient(chainCtx context.Context, opts ...Option) networkservice.NetworkServiceClient {
	o := &option{}
	for _, opt := range opts {
		opt(o)
	}
	return &dialClient{
		chainCtx:    chainCtx,
		dialOptions: o.dialOptions,
		dialTimeout: o.dialTimeout,
	}
}

func (d *dialClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	clientURL := clienturlctx.ClientURL(ctx)

	// If no clientURL, we have no work to do
	if clientURL == nil {
		return next.Client(ctx).Request(ctx, request, opts...)
	}

	cc, ccLoaded := clientconn.Load(ctx)

	// If the previousClientURL doesn't match the current clientURL, close the old connection,
	// which will trigger redial
	previousClientURL, urlLoaded := loadClientURL(ctx)
	if ccLoaded && urlLoaded && previousClientURL.String() != clientURL.String() {
		closer, ok := cc.(io.Closer)
		if ok {
			_ = closer.Close()
		}
		ccLoaded = false
		clientconn.Delete(ctx)
		deleteClientURL(ctx)
	}

	// If we have no current cc, dial
	if !ccLoaded {
		dialCtx := ctx
		if d.dialTimeout != 0 {
			dialCtx, _ = clock.FromContext(d.chainCtx).WithTimeout(dialCtx, d.dialTimeout)
		}
		cc, err := grpc.DialContext(dialCtx, grpcutils.URLToTarget(clientURL), d.dialOptions...)
		if err != nil {
			if cc != nil {
				_ = cc.Close()
			}
			return nil, errors.Wrapf(err, "failed to dial %s", grpcutils.URLToTarget(clientURL))
		}
		// store the cc because it will be needed by later chain elements like connect
		clientconn.Store(ctx, cc)
		conn, err := next.Client(ctx).Request(ctx, request)
		if err != nil {
			// Since we had to dial, if this requests fail, we should close the cc so that we don't leak connections
			// On *subsequent* connections, things should eventually be closed, if for no other reason than via timeout
			// but if this initial request fails, we need to close the grpc.ClientConn
			_ = cc.Close()
			clientconn.Delete(ctx)
			return nil, err
		}
		go func() {
			<-d.chainCtx.Done()
			_ = cc.Close()
		}()
		storeClientURL(ctx, metadata.IsClient(d), clientURL)
		return conn, nil
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (d *dialClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	_, err := next.Client(ctx).Close(ctx, conn, opts...)

	clientURL := clienturlctx.ClientURL(ctx)
	_, urlLoaded := loadClientURL(ctx)

	// If we either have no URL for the close, or didn't previously have a URL
	if clientURL == nil || !urlLoaded {
		return &emptypb.Empty{}, err
	}
	cc, loaded := clientconn.Load(ctx)
	closer, ok := cc.(io.Closer)
	if loaded && ok {
		_ = closer.Close()
	}
	clientconn.Delete(ctx)
	deleteClientURL(ctx)
	return &emptypb.Empty{}, err
}
