// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package expire

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/after"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type expireNSEServer struct {
	ctx           context.Context
	nseExpiration time.Duration
	afterFuncs    afterFuncMap
}

// NewNetworkServiceEndpointRegistryServer wraps passed NetworkServiceEndpointRegistryServer and monitor Network service endpoints
func NewNetworkServiceEndpointRegistryServer(ctx context.Context, nseExpiration time.Duration) registry.NetworkServiceEndpointRegistryServer {
	return &expireNSEServer{
		ctx:           ctx,
		nseExpiration: nseExpiration,
	}
}

func (n *expireNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	afterFunc, loaded := n.afterFuncs.Load(nse.Name)
	stopped := loaded && afterFunc.Stop()

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		if stopped {
			afterFunc.Resume()
		}
		return nil, err
	}

	expirationTime := time.Now().Add(n.nseExpiration)
	if resp.ExpirationTime != nil {
		if respExpirationTime := resp.ExpirationTime.AsTime().Local(); respExpirationTime.Before(expirationTime) {
			expirationTime = respExpirationTime
		}
	}
	resp.ExpirationTime = timestamppb.New(expirationTime)

	if loaded {
		afterFunc.Reset(expirationTime)
	} else {
		n.startAfterFunc(ctx, resp.Clone())
	}

	return resp, nil
}

func (n *expireNSEServer) startAfterFunc(ctx context.Context, nse *registry.NetworkServiceEndpoint) {
	logger := log.FromContext(ctx).WithField("expireNSEServer", "startAfterFunc")

	n.afterFuncs.Store(nse.Name, after.NewFunc(n.ctx, nse.ExpirationTime.AsTime().Local(), func() {
		n.afterFuncs.Delete(nse.Name)

		unregisterCtx, cancel := context.WithCancel(n.ctx)
		defer cancel()

		if _, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(unregisterCtx, nse); err != nil {
			logger.Errorf("failed to unregister expired endpoint: %s %s", nse.Name, err.Error())
		}
	}))
}

func (n *expireNSEServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (n *expireNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("expireNSEServer", "Unregister")

	afterFunc, ok := n.afterFuncs.LoadAndDelete(nse.Name)
	if !ok {
		logger.Warnf("endpoint has been already unregistered: %s", nse.Name)
		return new(empty.Empty), nil
	}
	afterFunc.Stop()

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
