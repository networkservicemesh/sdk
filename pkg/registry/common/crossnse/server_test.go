// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package crossnse_test define a tests for cross connect NSE chain element
package crossnse_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/crossnse"
)

type crossNSEMap struct {
	nses  sync.Map
	count int64
}

func (c *crossNSEMap) LoadOrStore(name string, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, bool) {
	val, loaded := c.nses.LoadOrStore(name, request)
	if !loaded {
		atomic.AddInt64(&c.count, 1)
	}
	return val.(*registry.NetworkServiceEndpoint), loaded
}

func (c *crossNSEMap) Delete(name string) {
	c.nses.Delete(name)
}

func TestCrossNSERegister(t *testing.T) {
	crossMap := &crossNSEMap{}
	server := crossnse.NewNetworkServiceRegistryServer(crossMap)

	reg, err := server.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: crossnse.CrossNSEName,
		Url:  "test",
	})
	require.Nil(t, err)
	require.Greater(t, len(reg.Name), len(crossnse.CrossNSEName))
	require.Equal(t, int64(1), crossMap.count)

	_, err = server.Unregister(context.Background(), reg)
	require.Nil(t, err)
}
func TestCrossNSERegisterInvalidURL(t *testing.T) {
	crossMap := &crossNSEMap{}
	server := crossnse.NewNetworkServiceRegistryServer(crossMap)

	req, err := server.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: crossnse.CrossNSEName,
		Url:  "ht% 20", // empty URL error
	})
	require.NotNil(t, err)
	require.Nil(t, req)
}
