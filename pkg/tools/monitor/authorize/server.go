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

// Package authorize provides authz checks for incoming or returning connections.
package authorize

import (
	"crypto/x509"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/tools/monitor/next"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
	"github.com/networkservicemesh/sdk/pkg/tools/spire"
)

type authorizeMonitorConnectionsServer struct {
	policies              policiesList
	spiffeIDConnectionMap *spire.SpiffeIDConnectionMap
}

// NewMonitorConnectionServer - returns a new authorization networkservicemesh.MonitorConnectionServer
func NewMonitorConnectionServer(opts ...Option) networkservice.MonitorConnectionServer {
	o := &options{
		policies:              policiesList{opa.WithServiceOwnConnectionPolicy()},
		spiffeIDConnectionMap: &spire.SpiffeIDConnectionMap{},
	}
	for _, opt := range opts {
		opt(o)
	}
	var s = &authorizeMonitorConnectionsServer{
		policies:              o.policies,
		spiffeIDConnectionMap: o.spiffeIDConnectionMap,
	}
	return s
}

// MonitorOpaInput - used to pass complex structure to monitor policies
type MonitorOpaInput struct {
	SpiffeIDConnectionMap map[string][]string           `json:"spiffe_id_connection_map"`
	PathSegments          []*networkservice.PathSegment `json:"path_segments"`
	ServiceSpiffeID       string                        `json:"service_spiffe_id"`
}

func (a *authorizeMonitorConnectionsServer) MonitorConnections(in *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	ctx := srv.Context()
	p, ok := peer.FromContext(ctx)
	var cert *x509.Certificate
	if ok {
		cert = opa.ParseX509Cert(p.AuthInfo)
	}
	var input MonitorOpaInput
	var spiffeID spiffeid.ID
	if cert != nil {
		spiffeID, _ = x509svid.IDFromCert(cert)
	}
	simpleMap := make(map[string][]string)
	a.spiffeIDConnectionMap.Range(
		func(k string, v []string) bool {
			simpleMap[k] = v
			return true
		},
	)

	input = MonitorOpaInput{
		ServiceSpiffeID:       spiffeID.String(),
		SpiffeIDConnectionMap: simpleMap,
		PathSegments:          in.PathSegments,
	}

	if err := a.policies.check(ctx, input); err != nil {
		return err
	}

	return next.MonitorConnectionServer(ctx).MonitorConnections(in, srv)
}
