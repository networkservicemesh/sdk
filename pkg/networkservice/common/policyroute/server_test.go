// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

package policyroute_test

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"
	"gopkg.in/yaml.v2"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/policyroute"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/fs"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const defaultPrefixesFileName = "policies.yaml"

// newPolicyRoutesGetter - is an object that can be dynamically updated.
func newPolicyRoutesGetter(ctx context.Context, configPath string) *policyRoutesGetter {
	p := &policyRoutesGetter{
		ctx: ctx,
	}
	p.policyRoutes.Store([]*networkservice.PolicyRoute{})
	updatePrefixes := func(bytes []byte) {
		if bytes == nil {
			p.policyRoutes.Store([]*networkservice.PolicyRoute{})
		}
		var source []*networkservice.PolicyRoute
		err := yaml.Unmarshal(bytes, &source)
		if err != nil {
			log.FromContext(ctx).Errorf("Cannot unmarshal policies, err: %v", err.Error())
			return
		}
		p.policyRoutes.Store(source)
	}
	updateCh := fs.WatchFile(p.ctx, configPath)
	updatePrefixes(<-updateCh)
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case update := <-updateCh:
				updatePrefixes(update)
			}
		}
	}()

	return p
}

func (p *policyRoutesGetter) GetPolicyRoutes() []*networkservice.PolicyRoute {
	return p.policyRoutes.Load().([]*networkservice.PolicyRoute)
}

type policyRoutesGetter struct {
	ctx          context.Context
	policyRoutes atomic.Value
}

func TestCheckReloadedPolicies(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	chainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Create policies
	policies := []*networkservice.PolicyRoute{
		{From: "172.16.2.201/24", Proto: "6", DstPort: "6666", Routes: []*networkservice.Route{{
			Prefix:  "172.16.3.0/24",
			NextHop: "172.16.2.200",
		}}},
		{Proto: "17", DstPort: "6666"},
		{Proto: "17", DstPort: "5555", Routes: []*networkservice.Route{{
			Prefix:  "2004::5/120",
			NextHop: "2004::6",
		}}},
	}

	// Write empty policies to file
	dir := filepath.Join(os.TempDir(), t.Name())
	defer func() { _ = os.RemoveAll(dir) }()
	require.NoError(t, os.MkdirAll(dir, os.ModePerm))
	configBytes, _ := yaml.Marshal(policies)
	configPath := filepath.Join(dir, defaultPrefixesFileName)
	require.NoError(t, os.WriteFile(configPath, []byte(""), os.ModePerm))

	getter := newPolicyRoutesGetter(chainCtx, configPath)

	server := chain.NewNetworkServiceServer(
		policyroute.NewServer(getter.GetPolicyRoutes),
	)

	// Write policies to file
	require.NoError(t, os.WriteFile(configPath, configBytes, os.ModePerm))

	// Initial request already has source Ip address
	srcIPAddr := "172.16.2.200/24"
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{SrcIpAddrs: []string{srcIPAddr}},
			},
		},
	}
	require.Eventually(t, func() bool {
		_, err := server.Request(context.Background(), req)
		require.NoError(t, err)
		ipCtx := req.GetConnection().GetContext().GetIpContext()
		return isEqual(ipCtx.GetPolicies(), policies) &&
			len(ipCtx.GetSrcIpAddrs()) == 2 &&
			ipCtx.GetSrcIpAddrs()[0] == srcIPAddr &&
			ipCtx.GetSrcIpAddrs()[1] == policies[0].GetFrom()
	}, time.Second, time.Millisecond*100)

	// Update policies - remove the first one
	policies = policies[1:]
	configBytes, _ = yaml.Marshal(policies)
	require.NoError(t, os.WriteFile(configPath, configBytes, os.ModePerm))

	require.Eventually(t, func() bool {
		_, err := server.Request(context.Background(), req)
		require.NoError(t, err)
		ipCtx := req.GetConnection().GetContext().GetIpContext()
		return isEqual(ipCtx.GetPolicies(), policies) &&
			len(ipCtx.GetSrcIpAddrs()) == 1 &&
			ipCtx.GetSrcIpAddrs()[0] == srcIPAddr
	}, time.Second, time.Millisecond*100)

	// Delete config file
	err := os.Remove(configPath)
	require.Nil(t, err)

	require.Eventually(t, func() bool {
		_, reqErr := server.Request(context.Background(), req)
		require.NoError(t, reqErr)
		ipCtx := req.GetConnection().GetContext().GetIpContext()
		return len(ipCtx.GetPolicies()) == 0 && len(ipCtx.GetSrcIpAddrs()) == 1
	}, time.Second, time.Millisecond*100)
}

func isEqual(p1, p2 []*networkservice.PolicyRoute) bool {
	if len(p1) != len(p2) {
		return false
	}
	for i := 0; i < len(p1); i++ {
		if !proto.Equal(p1[i], p2[i]) {
			return false
		}
	}
	return true
}
