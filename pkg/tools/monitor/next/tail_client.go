// Copyright (c) 2022 Cisco Systems, Inc.
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

package next

import (
	"context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

// tailMonitorConnectionClient is a simple implementation of networkservice.MonitorConnectionClient that is called at the end of a chain
// to insure that we never call a method on a nil object

type tailMonitorConnectionClient struct {
	networkservice.MonitorConnection_MonitorConnectionsClient
}

func (t *tailMonitorConnectionClient) MonitorConnections(ctx context.Context, in *networkservice.MonitorScopeSelector, opts ...grpc.CallOption) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	return t.MonitorConnection_MonitorConnectionsClient, nil
}
