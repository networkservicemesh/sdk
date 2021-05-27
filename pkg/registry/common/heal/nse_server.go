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

package heal

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type healNSEServer struct {
	ctx             context.Context
	onHeal          *registry.NetworkServiceEndpointRegistryServer
	ctxWrappers     ctxWrapperMap
	findCtxWrappers findCtxWrapperMap
}

func NewNetworkServiceEndpointRegistryServer(ctx context.Context, onHeal *registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	s := &healNSEServer{
		ctx:    ctx,
		onHeal: onHeal,
	}

	if onHeal == nil {
		s.onHeal = addressof.NetworkServiceEndpointRegistryServer(s)
	}

	return s
}

func (s *healNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	ctx = withRequestNSERestore(ctx, s.handleRestoreRequest)

	reg, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	if cw, loaded := s.ctxWrappers.LoadOrStore(reg.Name, new(ctxWrapper)); loaded {
		cw.lock.Lock()
		defer cw.lock.Unlock()

		if cw.cancel != nil {
			cw.cancel()
			cw.cancel = nil
		}
	}

	return reg, nil
}

func (s *healNSEServer) handleRestoreRequest(nse *registry.NetworkServiceEndpoint) {
	cw, loaded := s.ctxWrappers.Load(nse.Name)
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

	go s.restore(ctx, nse.Clone())
}

func (s *healNSEServer) restore(ctx context.Context, nse *registry.NetworkServiceEndpoint) {
	clockTime := clock.FromContext(ctx)
	logger := log.FromContext(ctx).WithField("healNSEServer", "restore")

	restoreCtx, cancel := clockTime.WithTimeout(ctx, time.Minute)
	defer cancel()

	logger.Infof("Started NSE restore: %s", nse.Name)
	for restoreCtx.Err() == nil {
		if _, err := (*s.onHeal).Register(restoreCtx, nse.Clone()); err != nil {
			logger.Warnf("Failed to restore NSE, retrying: %s %s", nse.Name, err.Error())
			continue
		}
		logger.Infof("NSE restored: %s", nse.Name)
		return
	}
	logger.Errorf("Failed to restore NSE: %s", nse.Name)

	logger.Infof("Unregistering NSE: %s", nse.Name)
	if _, err := (*s.onHeal).Unregister(ctx, nse.Clone()); err != nil {
		logger.Errorf("Failed to unregister NSE: %s %s", err.Error())
		return
	}
	logger.Infof("NSE unregistered: %s", nse.Name)
}

func (s *healNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	findID := uuid.New().String()
	fcw, _ := s.findCtxWrappers.LoadOrStore(findID, &findCtxWrapper{
		ctx:       server.Context(),
		restoreCh: make(chan struct{}),
		resultCh:  make(chan error),
	})

	server = streamcontext.NetworkServiceEndpointRegistryFindServer(withRequestNSERestoreFind(server.Context(), func() {
		s.handleRestoreFindRequest(findID)
	}), server)

	err := next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
	for err != nil {
		if _, ok := <-fcw.restoreCh; !ok {
			break
		}

		err = next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
		fcw.resultCh <- err
	}
	return err
}

func (s *healNSEServer) handleRestoreFindRequest(findID string) {
	fcw, loaded := s.findCtxWrappers.Load(findID)
	if !loaded {
		return
	}

	fcw.lock.Lock()
	defer fcw.lock.Unlock()

	if fcw.cancel != nil {
		fcw.cancel() // TODO: such case should be invalid, we don't need to restart healing
	}

	var ctx context.Context
	ctx, fcw.cancel = context.WithCancel(fcw.ctx)

	go s.restoreFind(ctx, fcw.restoreCh, fcw.resultCh)
}

func (s *healNSEServer) restoreFind(ctx context.Context, restoreCh chan<- struct{}, resultCh <-chan error) {
	clockTime := clock.FromContext(ctx)
	logger := log.FromContext(ctx).WithField("healNSEServer", "restoreFind")

	restoreCtx, cancel := clockTime.WithTimeout(ctx, time.Minute)
	defer cancel()

	logger.Infof("Started NSE Find restore")
loop:
	for {
		select {
		case <-restoreCtx.Done():
			break loop
		case restoreCh <- struct{}{}:
		}

		select {
		case <-restoreCtx.Done():
			break loop
		case <-clockTime.After(10 * time.Second): // TODO: option
		case err := <-resultCh:
			if err != nil {
				logger.Warnf("Failed to restore NSE Find, retrying: %s", err.Error())
				continue
			}
		}

		logger.Infof("NSE Find restored")
		return
	}

	logger.Errorf("Failed to restore NSE Find")
	close(restoreCh)
}

func (s *healNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	cw, loaded := s.ctxWrappers.LoadAndDelete(nse.Name)
	if !loaded {
		return new(empty.Empty), nil
	}

	cw.lock.Lock()

	if cw.cancel != nil {
		cw.cancel()
		cw.cancel = nil
	}

	cw.lock.Unlock()

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
