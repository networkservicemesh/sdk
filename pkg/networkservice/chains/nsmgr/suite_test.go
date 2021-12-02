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

// Package nsmgr_test define a tests for NSMGR chain element.
package nsmgr_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/replacelabels"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	nodesCount = 7
)

type nsmgrSuite struct {
	suite.Suite

	domain           *sandbox.Domain
	nsRegistryClient registry.NetworkServiceRegistryClient
}

func TestNsmgr(t *testing.T) {
	suite.Run(t, new(nsmgrSuite))
}

func (s *nsmgrSuite) SetupSuite() {
	t := s.T()

	ctx, cancel := context.WithCancel(context.Background())

	// Call cleanup when tests complete
	t.Cleanup(func() {
		cancel()
		goleak.VerifyNone(s.T())
	})

	// Create default domain with nodesCount nodes, which will be enough for any test
	s.domain = sandbox.NewBuilder(ctx, t).
		SetNodesCount(nodesCount).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		Build()

	s.nsRegistryClient = s.domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)
}

func (s *nsmgrSuite) Test_PassThroughRemoteUsecase() {
	t := s.T()

	log.EnableTracing(true)
	logrus.SetLevel(logrus.TraceLevel)
	// logrus.SetOutput(io.Discard)

	t.Cleanup(func() {
		log.EnableTracing(false)
		logrus.SetLevel(logrus.InfoLevel)
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*400)
	defer cancel()

	start := time.Now()
	counterClose := new(count.Server)

	nsReg := linearNS(nodesCount)
	nsReg, err := s.nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	var nseRegs [nodesCount]*registry.NetworkServiceEndpoint
	var nses [nodesCount]*sandbox.EndpointEntry
	for i := range nseRegs {
		nseRegs[i], nses[i] = newPassThroughEndpoint(
			ctx,
			s.domain.Nodes[i],
			map[string]string{
				step: fmt.Sprintf("%v", i),
			},
			fmt.Sprintf("%v", i),
			nsReg.Name,
			i != nodesCount-1,
			counterClose,
		)
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.Name)
	// Request
	conn, err := nsc.Request(ctx, request)

	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 5
	// Path length from NSE client to other remote endpoint is 8
	require.Equal(t, 6*(nodesCount-1)+4, len(conn.Path.PathSegments))
	for i := 0; i < len(nseRegs); i++ {
		require.Contains(t, nseRegs[i].Name, conn.Path.PathSegments[i*6+3].Name)
	}

	// Refresh
	for i := 0; i < 5; i++ {
		start1 := time.Now()

		request.Connection = conn.Clone()
		elapsed1 := time.Since(start1)
		fmt.Printf("conn.Clone. Elapsed time: %v\n", elapsed1)

		conn, err = nsc.Request(ctx, request)
		require.NoError(t, err)
	}

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, nodesCount, counterClose.Closes())

	// Endpoint unregister
	for i, nseReg := range nseRegs {
		_, err = nses[i].Unregister(ctx, nseReg)
		require.NoError(t, err)
	}
	elapsed := time.Since(start)
	fmt.Printf("Test_PassThroughRemoteUsecase. Elapsed time: %v\n", elapsed)
}

const (
	step   = "step"
	labelA = "label_a"
	labelB = "label_b"
)

func linearNS(hopsCount int) *registry.NetworkService {
	matches := make([]*registry.Match, 0)

	for i := 1; i < hopsCount; i++ {
		match := &registry.Match{
			SourceSelector: map[string]string{
				step: fmt.Sprintf("%v", i-1),
			},
			Routes: []*registry.Destination{
				{
					DestinationSelector: map[string]string{
						step: fmt.Sprintf("%v", i),
					},
				},
			},
		}

		matches = append(matches, match)
	}

	if hopsCount > 1 {
		// match with empty source selector must be the last
		match := &registry.Match{
			Routes: []*registry.Destination{
				{
					DestinationSelector: map[string]string{
						step: fmt.Sprintf("%v", 0),
					},
				},
			},
		}
		matches = append(matches, match)
	}

	return &registry.NetworkService{
		Name:    "test-network-service-linear",
		Matches: matches,
	}
}

func additionalFunctionalityChain(ctx context.Context, clientURL *url.URL, clientName string, labels map[string]string) []networkservice.NetworkServiceServer {
	return []networkservice.NetworkServiceServer{
		chain.NewNetworkServiceServer(
			clienturl.NewServer(clientURL),
			connect.NewServer(
				client.NewClient(
					ctx,
					client.WithName(fmt.Sprintf("endpoint-client-%v", clientName)),
					client.WithAdditionalFunctionality(
						mechanismtranslation.NewClient(),
						replacelabels.NewClient(labels),
						kernel.NewClient(),
					),
					client.WithDialOptions(sandbox.DialOptions()...),
					client.WithDialTimeout(sandbox.DialTimeout),
					client.WithoutRefresh(),
				),
			),
		),
	}
}

func newPassThroughEndpoint(
	ctx context.Context,
	node *sandbox.Node,
	labels map[string]string,
	name, nsRegName string,
	hasClientFunctionality bool,
	counter networkservice.NetworkServiceServer,
) (*registry.NetworkServiceEndpoint, *sandbox.EndpointEntry) {
	nseReg := &registry.NetworkServiceEndpoint{
		Name:                fmt.Sprintf("endpoint-%v", name),
		NetworkServiceNames: []string{nsRegName},
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			nsRegName: {
				Labels: labels,
			},
		},
	}

	var additionalFunctionality []networkservice.NetworkServiceServer
	if hasClientFunctionality {
		additionalFunctionality = additionalFunctionalityChain(ctx, node.NSMgr.URL, name, labels)
	}

	if counter != nil {
		additionalFunctionality = append(additionalFunctionality, counter)
	}

	return nseReg, node.NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, additionalFunctionality...)
}
