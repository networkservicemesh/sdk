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

package nsmgr_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	restartTimeout = 250 * time.Millisecond
	expireTimeout  = 3 * time.Second
)

type restartable interface {
	Restart()
	Cancel()
}

type sample struct {
	name     string
	elements []restartable
	remote   bool
}

/* RESTARTS */
/* The following examples represent a situation when some 2 elements were restarted */
func getRestartsSamples(nodes []*sandbox.Node, reg restartable, nses []restartable) []sample {
	return mergeSlices(
		getManagerManagerSamples(nodes),
		getManagerForwarderSamples(nodes),
		getManagerEndpointSamples(nodes, nses),
		getManagerRegistrySamples(nodes, reg),
		getForwarderForwarderSamples(nodes),
		getForwarderEndpointSamples(nodes, nses),

		changeBreakOrder(getManagerManagerSamples(nodes)),
		changeBreakOrder(getManagerRegistrySamples(nodes, reg)),
		changeBreakOrder(getForwarderForwarderSamples(nodes)),
		changeBreakOrder(getForwarderEndpointSamples(nodes, nses)),
		changeBreakOrder(getForwarderRegistrySamples(nodes, reg)),
		changeBreakOrder(getEndpointRegistrySamples(reg, nses)),
	)
}
func getManagerManagerSamples(nodes []*sandbox.Node) []sample {
	return []sample{
		{
			name:   "Local Mgr Remote Mgr",
			remote: true,
			elements: []restartable{
				nodes[0].NSMgr,
				nodes[1].NSMgr,
			},
		},
	}
}
func getManagerForwarderSamples(nodes []*sandbox.Node) []sample {
	fwdLocalName := getFwdName(nodes[0])
	fwdRemoteName := getFwdName(nodes[1])

	return []sample{
		{
			name: "Local Mgr Local Fwd",
			elements: []restartable{
				nodes[0].NSMgr,
				nodes[0].Forwarders[fwdLocalName],
			},
		},
		{
			name:   "Local Mgr Remote Fwd",
			remote: true,
			elements: []restartable{
				nodes[0].NSMgr,
				nodes[1].Forwarders[fwdRemoteName],
			},
		},
		{
			name:   "Remote Mgr Local Fwd",
			remote: true,
			elements: []restartable{
				nodes[1].NSMgr,
				nodes[0].Forwarders[fwdLocalName],
			},
		},
		{
			name:   "Remote Mgr Remote Fwd",
			remote: true,
			elements: []restartable{
				nodes[1].NSMgr,
				nodes[1].Forwarders[fwdRemoteName],
			},
		},
	}
}
func getManagerEndpointSamples(nodes []*sandbox.Node, nses []restartable) []sample {
	return []sample{
		{
			name: "Local Mgr Local Endpoint",
			elements: []restartable{
				nodes[0].NSMgr,
				nses[0],
			},
		},
		{
			name:   "Local Mgr Remote Endpoint",
			remote: true,
			elements: []restartable{
				nodes[0].NSMgr,
				nses[1],
			},
		},
		{
			name:   "Remote Mgr Remote Endpoint",
			remote: true,
			elements: []restartable{
				nodes[1].NSMgr,
				nses[1],
			},
		},
	}
}
func getManagerRegistrySamples(nodes []*sandbox.Node, reg restartable) []sample {
	return []sample{
		{
			name: "Local Mgr Registry",
			elements: []restartable{
				nodes[0].NSMgr,
				reg,
			},
		},
		{
			name:   "Local Mgr Registry Remote",
			remote: true,
			elements: []restartable{
				nodes[0].NSMgr,
				reg,
			},
		},
		{
			name:   "Remote Mgr Registry Remote",
			remote: true,
			elements: []restartable{
				nodes[1].NSMgr,
				reg,
			},
		},
	}
}
func getForwarderForwarderSamples(nodes []*sandbox.Node) []sample {
	fwdLocalName := getFwdName(nodes[0])
	fwdRemoteName := getFwdName(nodes[1])

	return []sample{
		{
			name:   "Local Fwd Remote Fwd",
			remote: true,
			elements: []restartable{
				nodes[0].Forwarders[fwdLocalName],
				nodes[1].Forwarders[fwdRemoteName],
			},
		},
	}
}
func getForwarderEndpointSamples(nodes []*sandbox.Node, nses []restartable) []sample {
	fwdLocalName := getFwdName(nodes[0])
	fwdRemoteName := getFwdName(nodes[1])

	return []sample{
		{
			name: "Local Fwd Local Endpoint",
			elements: []restartable{
				nodes[0].Forwarders[fwdLocalName],
				nses[0],
			},
		},
		{
			name:   "Local Fwd Remote Endpoint",
			remote: true,
			elements: []restartable{
				nodes[0].Forwarders[fwdLocalName],
				nses[1],
			},
		},
		{
			name:   "Remote Fwd Remote Endpoint",
			remote: true,
			elements: []restartable{
				nodes[1].Forwarders[fwdRemoteName],
				nses[1],
			},
		},
	}
}
func getForwarderRegistrySamples(nodes []*sandbox.Node, reg restartable) []sample {
	fwdLocalName := getFwdName(nodes[0])
	fwdRemoteName := getFwdName(nodes[1])
	return []sample{
		{
			name: "Local Fwd Registry",
			elements: []restartable{
				nodes[0].Forwarders[fwdLocalName],
				reg,
			},
		},
		{
			name:   "Local Fwd Registry Remote",
			remote: true,
			elements: []restartable{
				nodes[0].Forwarders[fwdLocalName],
				reg,
			},
		},
		{
			name:   "Remote Fwd Registry Remote",
			remote: true,
			elements: []restartable{
				nodes[1].Forwarders[fwdRemoteName],
				reg,
			},
		},
	}
}
func getEndpointRegistrySamples(reg restartable, nses []restartable) []sample {
	return []sample{
		{
			name: "Local Endpoint Registry",
			elements: []restartable{
				nses[0],
				reg,
			},
		},
		{
			name:   "Remote Endpoint Registry",
			remote: true,
			elements: []restartable{
				nses[1],
				reg,
			},
		},
	}
}

/* DEATHS */
/* The following examples represent a situation when 1 or 2 elements died and were recreated */
func getDeathSamples(ctx context.Context, nodes []*sandbox.Node, reg restartable, nseInfos []*nseDeathInfo) []sample {
	return mergeSlices(
		getManagerForwarderDeathSamples(ctx, nodes),
		getManagerEndpointDeathSamples(ctx, nodes, nseInfos),
		getForwarderEndpointDeathSamples(ctx, nodes, nseInfos),
		getForwarderForwarderDeathSamples(ctx, nodes),

		changeBreakOrder(getForwarderEndpointDeathSamples(ctx, nodes, nseInfos)),
		changeBreakOrder(getForwarderForwarderDeathSamples(ctx, nodes)),
		changeBreakOrder(getForwarderDeathRegistrySamples(ctx, nodes, reg)),
	)
}
func getManagerForwarderDeathSamples(ctx context.Context, nodes []*sandbox.Node) []sample {
	return []sample{
		{
			name: "Local Mgr Local Fwd Death",
			elements: []restartable{
				nodes[0].NSMgr,
				newForwarderDeath(ctx, nodes[0]),
			},
		},
		{
			name:   "Local Mgr Remote Fwd Death",
			remote: true,
			elements: []restartable{
				nodes[0].NSMgr,
				newForwarderDeath(ctx, nodes[1]),
			},
		},
		{
			name:   "Remote Mgr Local Fwd Death",
			remote: true,
			elements: []restartable{
				nodes[1].NSMgr,
				newForwarderDeath(ctx, nodes[0]),
			},
		},
		{
			name:   "Remote Mgr Remote Fwd Death",
			remote: true,
			elements: []restartable{
				nodes[1].NSMgr,
				newForwarderDeath(ctx, nodes[1]),
			},
		},
	}
}
func getManagerEndpointDeathSamples(ctx context.Context, nodes []*sandbox.Node, nses []*nseDeathInfo) []sample {
	return []sample{
		{
			name: "Local Mgr Local Endpoint Death",
			elements: []restartable{
				nodes[0].NSMgr,
				newEndpointDeath(ctx, nodes[0], nses[0]),
			},
		},
		{
			name:   "Local Mgr Remote Endpoint Death",
			remote: true,
			elements: []restartable{
				nodes[0].NSMgr,
				newEndpointDeath(ctx, nodes[1], nses[1]),
			},
		},
		{
			name:   "Remote Mgr Remote Endpoint Death",
			remote: true,
			elements: []restartable{
				nodes[1].NSMgr,
				newEndpointDeath(ctx, nodes[1], nses[1]),
			},
		},
	}
}
func getForwarderEndpointDeathSamples(ctx context.Context, nodes []*sandbox.Node, nses []*nseDeathInfo) []sample {
	return []sample{
		{
			name: "Local Fwd Local Endpoint Death",
			elements: []restartable{
				actualForwarder(ctx, nodes[0]),
				newEndpointDeath(ctx, nodes[0], nses[0]),
			},
		},
		{
			name:   "Local Fwd Remote Endpoint Death",
			remote: true,
			elements: []restartable{
				actualForwarder(ctx, nodes[0]),
				newEndpointDeath(ctx, nodes[1], nses[1]),
			},
		},
		{
			name:   "Remote Fwd Remote Endpoint Death",
			remote: true,
			elements: []restartable{
				actualForwarder(ctx, nodes[1]),
				newEndpointDeath(ctx, nodes[1], nses[1]),
			},
		},
		{
			name: "Local Fwd Death Local Endpoint Death",
			elements: []restartable{
				newForwarderDeath(ctx, nodes[0]),
				newEndpointDeath(ctx, nodes[0], nses[0]),
			},
		},
		{
			name:   "Local Fwd Death Remote Endpoint Death",
			remote: true,
			elements: []restartable{
				newForwarderDeath(ctx, nodes[0]),
				newEndpointDeath(ctx, nodes[1], nses[1]),
			},
		},
		{
			name:   "Remote Fwd Death Remote Endpoint Death",
			remote: true,
			elements: []restartable{
				newForwarderDeath(ctx, nodes[1]),
				newEndpointDeath(ctx, nodes[1], nses[1]),
			},
		},
	}
}
func getForwarderDeathRegistrySamples(ctx context.Context, nodes []*sandbox.Node, reg restartable) []sample {
	return []sample{
		{
			name: "Local Fwd Death Registry",
			elements: []restartable{
				newForwarderDeath(ctx, nodes[0]),
				reg,
			},
		},
		{
			name:   "Local Fwd Death Registry Remote",
			remote: true,
			elements: []restartable{
				newForwarderDeath(ctx, nodes[0]),
				reg,
			},
		},
		{
			name:   "Remote Fwd Death Registry Remote",
			remote: true,
			elements: []restartable{
				newForwarderDeath(ctx, nodes[1]),
				reg,
			},
		},
	}
}
func getForwarderForwarderDeathSamples(ctx context.Context, nodes []*sandbox.Node) []sample {
	return []sample{
		{
			name: "Local Fwd Death Remote Fwd Death",
			elements: []restartable{
				newForwarderDeath(ctx, nodes[0]),
				newForwarderDeath(ctx, nodes[1]),
			},
		},
	}
}

func TestNSMGR_Heal_Advanced(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout*50)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(2).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetRegistryExpiryDuration(expireTimeout).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsRegLocal, err := nsRegistryClient.Register(ctx,
		&registry.NetworkService{Name: "nslocal"})
	require.NoError(t, err)

	nsRegRemote, err := nsRegistryClient.Register(ctx,
		&registry.NetworkService{Name: "nsremote"})
	require.NoError(t, err)

	counterLocal := new(count.Server)
	counterRemote := new(count.Server)

	var nse0 restartable = domain.Nodes[0].NewEndpoint(ctx, &registry.NetworkServiceEndpoint{
		Name:                "nse-local",
		NetworkServiceNames: []string{nsRegLocal.Name},
	}, sandbox.GenerateTestToken, counterLocal)

	var nse1 restartable = domain.Nodes[1].NewEndpoint(ctx, &registry.NetworkServiceEndpoint{
		Name:                "nse-remote",
		NetworkServiceNames: []string{nsRegRemote.Name},
	}, sandbox.GenerateTestToken, counterRemote)

	var samples = mergeSlices(
		getRestartsSamples(domain.Nodes, domain.Registry, []restartable{nse0, nse1}),
		getDeathSamples(ctx, domain.Nodes, domain.Registry, []*nseDeathInfo{
			{nse: &nse0, nsName: "nslocal", counter: counterLocal},
			{nse: &nse1, nsName: "nsremote", counter: counterRemote},
		}),
	)
	iter := 0
	for _, sample := range samples {
		elements := sample.elements

		nsReg := nsRegLocal
		counter := counterLocal
		if sample.remote {
			nsReg = nsRegRemote
			counter = counterRemote
		}
		prevUniqueRequests := counter.UniqueRequests()
		prevRequests := counter.Requests()

		t.Run(sample.name, func(t *testing.T) {
			request := defaultRequest(nsReg.Name)
			request.Connection.Id = strconv.Itoa(iter)

			nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

			conn, err := nsc.Request(ctx, request.Clone())
			require.NoError(t, err)
			require.Equal(t, prevUniqueRequests+1, counter.UniqueRequests())

			// Break the connection
			for _, elem := range elements {
				elem.Cancel()
			}
			// All broken elements are unavailable at the same time
			time.Sleep(restartTimeout)
			for _, elem := range elements {
				elem.Restart()
				// One element is already available, the other is not
				time.Sleep(restartTimeout / 2)
			}

			// Wait for the connection to heal
			require.Eventually(t,
				func() bool { return prevRequests+2 == counter.Requests() },
				timeout*10, tick)

			// Check refresh
			request.Connection = conn
			_, err = nsc.Request(ctx, request.Clone())
			require.NoError(t, err)
			require.Equal(t, prevRequests+3, counter.Requests())

			// Close the connection
			prevCloses := counter.UniqueCloses()
			_, err = nsc.Close(ctx, conn)
			require.NoError(t, err)
			require.Equal(t, prevCloses+1, counter.Closes())

			iter++
		})
	}
}

func getFwdName(node *sandbox.Node) string {
	for forwarder := range node.Forwarders {
		return forwarder
	}
	return ""
}

func changeBreakOrder(samples []sample) []sample {
	for _, sample := range samples {
		sample.elements[0], sample.elements[1] = sample.elements[1], sample.elements[0]
	}
	return samples
}

func mergeSlices(args ...[]sample) []sample {
	mergedSlice := make([]sample, 0)
	for _, slice := range args {
		mergedSlice = append(mergedSlice, slice...)
	}

	return mergedSlice
}

// Create a new Forwarder that is always up to date (regardless of previous deaths)
func actualForwarder(ctx context.Context, node *sandbox.Node) *actualFwd {
	return &actualFwd{
		ctx:  ctx,
		node: node,
	}
}

type actualFwd struct {
	restartable

	ctx  context.Context
	node *sandbox.Node
}

func (f *actualFwd) Restart() {
	if f.restartable == nil {
		f.node.Forwarders[getFwdName(f.node)].Restart()
	} else {
		f.restartable.Restart()
	}
}

func (f *actualFwd) Cancel() {
	f.restartable = f.node.Forwarders[getFwdName(f.node)]
	f.restartable.Cancel()
}

// Create a new Forwarder to override Restart method for Death case
func newForwarderDeath(ctx context.Context, node *sandbox.Node) *forwarderDeath {
	return &forwarderDeath{
		ctx:  ctx,
		node: node,
	}
}

type forwarderDeath struct {
	restartable

	ctx  context.Context
	node *sandbox.Node
}

func (f *forwarderDeath) Restart() {
	forwarderReg := &registry.NetworkServiceEndpoint{
		Name:                sandbox.UniqueName("forwarder"),
		NetworkServiceNames: []string{"forwarder"},
	}
	f.node.NewForwarder(f.ctx, forwarderReg, sandbox.GenerateTestToken)
}

func (f *forwarderDeath) Cancel() {
	f.node.Forwarders[getFwdName(f.node)].Cancel()
}

// Create a new NSE to override Restart method for Death case
type nseDeathInfo struct {
	nse     *restartable
	nsName  string
	counter networkservice.NetworkServiceServer
}

func newEndpointDeath(ctx context.Context, node *sandbox.Node, info *nseDeathInfo) *endpointDeath {
	return &endpointDeath{
		nse:  info.nse,
		ctx:  ctx,
		node: node,

		nsName:  info.nsName,
		counter: info.counter,
	}
}

type endpointDeath struct {
	nse *restartable

	ctx  context.Context
	node *sandbox.Node

	nsName  string
	counter networkservice.NetworkServiceServer
}

func (f *endpointDeath) Restart() {
	*f.nse = f.node.NewEndpoint(f.ctx, &registry.NetworkServiceEndpoint{
		Name:                sandbox.UniqueName("nse"),
		NetworkServiceNames: []string{f.nsName},
	}, sandbox.GenerateTestToken, f.counter)
}

func (f *endpointDeath) Cancel() {
	(*f.nse).Cancel()
}
