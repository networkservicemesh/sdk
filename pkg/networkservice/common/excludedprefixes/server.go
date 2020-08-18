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
	"io/ioutil"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/fswatcher"
	"github.com/networkservicemesh/sdk/pkg/tools/prefixpool"
)

type excludedPrefixesServer struct {
	ctx        context.Context
	prefixPool atomic.Value
	once       sync.Once
	configPath string
}

func (eps *excludedPrefixesServer) init() {
	logger := trace.Log(eps.ctx)
	onUpdate := func(event fsnotify.Event) error {
		configPath, err := filepath.Abs(event.Name)
		if err != nil {
			logger.Warnf("error getting config path from the event, try to use the stored path")
			return eps.updatePrefixPool(eps.configPath)
		}
		return eps.updatePrefixPool(configPath)
	}
	if err := eps.updatePrefixPool(eps.configPath); err != nil {
		pool, _ := prefixpool.New()
		eps.prefixPool.Store(pool)
	}
	go func() {
		err := fswatcher.WatchOn(eps.ctx, onUpdate, eps.configPath)
		if err != nil {
			logger.Errorf("An error during watch file: %v", err.Error())
		}
	}()
}

func (eps *excludedPrefixesServer) updatePrefixPool(configPath string) error {
	logger := trace.Log(eps.ctx)
	bytes, err := ioutil.ReadFile(filepath.Clean(configPath))
	if err != nil {
		logger.Errorf("An error during read file %v, error: %v", configPath, err.Error())
		return err
	}
	source := struct {
		Prefixes []string
	}{}
	err = yaml.Unmarshal(bytes, &source)
	if err != nil {
		logger.Errorf("Can not create unmarshal prefixes, err: %v", err.Error())
		return err
	}
	pool, err := prefixpool.New(source.Prefixes...)
	if err != nil {
		logger.Errorf("Can not create prefixpool with prefixes: %+v, err: %v", source.Prefixes, err.Error())
		return err
	}
	eps.prefixPool.Store(pool)
	return nil
}

// Note: request.Connection and Connection.Context should not be nil
func (eps *excludedPrefixesServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	eps.once.Do(eps.init)
	logger := trace.Log(ctx)

	conn := request.GetConnection()
	if conn.GetContext().GetIpContext() == nil {
		conn.Context.IpContext = &networkservice.IPContext{}
	}
	prefixes := eps.prefixPool.Load().(*prefixpool.PrefixPool).GetPrefixes()
	logger.Infof("ExcludedPrefixesService: adding excluded prefixes to connection: %v", prefixes)
	ipCtx := conn.GetContext().GetIpContext()
	ipCtx.ExcludedPrefixes = removeDuplicates(append(ipCtx.GetExcludedPrefixes(), prefixes...))

	return next.Server(ctx).Request(ctx, request)
}

func (eps *excludedPrefixesServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, connection)
}

// NewServer -  creates a networkservice.NetworkServiceServer chain element that can read excluded prefixes from config
// map and add them to request to avoid repeated usage.
// Note: request.Connection and Connection.Context should not be nil when calling Request
func NewServer(ctx context.Context, setters ...ServerOption) networkservice.NetworkServiceServer {
	server := &excludedPrefixesServer{
		configPath: prefixesFilePathDefault,
		ctx:        ctx,
	}
	for _, setter := range setters {
		setter(server)
	}

	return server
}
