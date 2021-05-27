// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// SPDX-Licens-Identifier: Apache-2.0
//
// Licensd under the Apache Licens, Version 2.0 (the "Licens");
// you may not use this file except in compliance with the Licens.
// You may obtain a copy of the Licens at:
//
//     http://www.apache.org/licenss/LICENS-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the Licens is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the Licens for the specific language governing permissions and
// limitations under the Licens.

package heal

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type OnHealNSFunc func(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error)

type healNSServer struct {
	ctx           context.Context
	onHeal        OnHealNSFunc
	nsCtxWrappers nsCtxWrapperMap
}

type nsCtxWrapper struct {
	cancel context.CancelFunc
	lock   sync.Mutex
}

func NewNetworkServiceRegistryServer(ctx context.Context, onHeal OnHealNSFunc) registry.NetworkServiceRegistryServer {
	rv := &healNSServer{
		ctx:    ctx,
		onHeal: onHeal,
	}

	if rv.onHeal == nil {
		rv.onHeal = rv.Register
	}

	return rv
}

func (s *healNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	ctx = withRequestNSRestore(ctx, s.handleRestoreRequest)

	reg, err := next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
	if err != nil {
		return nil, err
	}

	if cw, loaded := s.nsCtxWrappers.LoadOrStore(reg.Name, new(nsCtxWrapper)); loaded {
		cw.lock.Lock()
		defer cw.lock.Unlock()

		if cw.cancel != nil {
			cw.cancel()
			cw.cancel = nil
		}
	}

	return reg, nil
}

func (s *healNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server) // TODO: don't close stream on healing
}

func (s *healNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	cw, loaded := s.nsCtxWrappers.LoadAndDelete(ns.Name)
	if !loaded {
		return new(empty.Empty), nil
	}

	cw.lock.Lock()

	if cw.cancel != nil {
		cw.cancel()
		cw.cancel = nil
	}

	cw.lock.Unlock()

	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}

func (s *healNSServer) handleRestoreRequest(ns *registry.NetworkService) {
	cw, loaded := s.nsCtxWrappers.Load(ns.Name)
	if !loaded {
		return
	}

	cw.lock.Lock()
	defer cw.lock.Unlock()

	if cw.cancel != nil {
		cw.cancel() // TODO: such case should be invalid, we don't need to restart healing
	}

	var ctx context.Context
	ctx, cw.cancel = context.WithCancel(s.ctx)

	go s.restore(ctx, ns.Clone())
}

func (s *healNSServer) restore(ctx context.Context, ns *registry.NetworkService) {
	logger := log.FromContext(ctx).WithField("healNSServer", "restore")

	logger.Infof("Started NS restore: %v", ns.Name)
	for ctx.Err() == nil {
		if _, err := s.onHeal(ctx, ns.Clone()); err == nil {
			logger.Infof("NS restored: %v", ns.Name)
			return
		}
		logger.Warnf("Failed to restore NS, retrying: %v", ns.Name)
	}
	logger.Errorf("Failed to restore NS: %v", ns.Name)
}
