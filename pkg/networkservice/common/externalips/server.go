// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package externalips provides to the context on Request or Close possible to resolve external IP to internal or vise versa
package externalips

import (
	"context"
	"net"
	"sync/atomic"

	"github.com/edwarnicke/genericsync"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/fs"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type externalIPsServer struct {
	internalToExternalMap atomic.Value
	externalToInternalMap atomic.Value
	updateCh              <-chan map[string]string
	chainCtx              context.Context
}

func (e *externalIPsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	ctx = withExternalReplacer(ctx, replaceFunc(e.externalToInternalMap))
	ctx = withInternalReplacer(ctx, replaceFunc(e.internalToExternalMap))

	return next.Server(ctx).Request(ctx, request)
}

func (e *externalIPsServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	ctx = withExternalReplacer(ctx, replaceFunc(e.externalToInternalMap))
	ctx = withInternalReplacer(ctx, replaceFunc(e.internalToExternalMap))

	return next.Server(ctx).Close(ctx, connection)
}

// NewServer creates networkservice.NetworkServiceServer which provides to context possible to resolve internal IP to external or vise versa.
// By default watches file by DefaultFilePath.
func NewServer(chainCtx context.Context, options ...Option) networkservice.NetworkServiceServer {
	result := &externalIPsServer{
		chainCtx: chainCtx,
	}
	result.externalToInternalMap.Store(new(genericsync.Map[string, string]))
	result.internalToExternalMap.Store(new(genericsync.Map[string, string]))
	for _, o := range options {
		o(result)
	}
	if result.updateCh == nil {
		result.updateCh = monitorMapFromFile(chainCtx, DefaultFilePath)
	}
	go func() {
		logger := log.FromContext(chainCtx).WithField("externalIPsServer", "build")
		for {
			select {
			case <-chainCtx.Done():
				return
			case update := <-result.updateCh:
				if err := result.build(update); err != nil {
					logger.Error(err.Error())
				} else {
					logger.Info("rebuilt internal and external ips map")
				}
			}
		}
	}()
	return result
}

func replaceFunc(v atomic.Value) func(ip net.IP) net.IP {
	m := v.Load().(*genericsync.Map[string, string])
	return func(ip net.IP) net.IP {
		key := ip.String()
		value, _ := m.Load(key)
		return net.ParseIP(value)
	}
}

func (e *externalIPsServer) build(ips map[string]string) error {
	validate := func(s string) error {
		if ip := net.ParseIP(s); ip == nil {
			return errors.Errorf("%v is not IP", ip)
		}
		return nil
	}
	for k, v := range ips {
		if err := validate(k); err != nil {
			return err
		}
		if err := validate(v); err != nil {
			return err
		}
	}
	internalIPs, externalIPs := new(genericsync.Map[string, string]), new(genericsync.Map[string, string])
	for k, v := range ips {
		internalIPs.Store(k, v)
		externalIPs.Store(v, k)
	}
	e.internalToExternalMap.Store(internalIPs)
	e.externalToInternalMap.Store(externalIPs)
	return nil
}

func monitorMapFromFile(ctx context.Context, path string) <-chan map[string]string {
	ch := make(chan map[string]string)
	go func() {
		for bytes := range fs.WatchFile(ctx, path) {
			var m map[string]string
			err := yaml.Unmarshal(bytes, &m)
			if err != nil {
				log.FromContext(ctx).WithField("externalIPsServer", "yaml.Unmarshal").Error(err.Error())
				continue
			}
			select {
			case ch <- m:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}
