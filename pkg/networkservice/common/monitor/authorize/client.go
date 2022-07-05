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

package authorize

import (
	"context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor/next"
)

type authorizeMonitorConnectionsClient struct {
	policies policiesList
}

// NewMonitorConnectionsClient - returns a new authorization NewMonitorConnectionsClient
// Authorize client checks rigiht side of path.
func NewMonitorConnectionsClient(opts ...Option) networkservice.MonitorConnectionClient {
	var result = &authorizeMonitorConnectionsClient{
		policies: []Policy{},
	}

	for _, o := range opts {
		o.apply(&result.policies)
	}

	return result
}

func (a *authorizeMonitorConnectionsClient) MonitorConnections(ctx context.Context, in *networkservice.MonitorScopeSelector, opts ...grpc.CallOption) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	srv, err := next.MonitorConnectionClient(ctx).MonitorConnections(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	pathSegments := in.GetPathSegments()
	var path = &networkservice.Path{
		PathSegments: pathSegments[:len(pathSegments)-1],
	}
	if err := a.policies.check(ctx, path); err != nil {
		return nil, err
	}

	return srv, nil
}
