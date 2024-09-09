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

package trace

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace/traceconcise"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace/traceverbose"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type traceServer struct {
	verbose networkservice.NetworkServiceServer
	concise networkservice.NetworkServiceServer
}

// NewNetworkServiceServer - wraps tracing around the supplied traced.
func NewNetworkServiceServer(traced networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return &traceServer{
		verbose: traceverbose.NewNetworkServiceServer(traced),
		concise: traceconcise.NewNetworkServiceServer(traced),
	}
}

func (t *traceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Request(ctx, request)
	}
	return t.concise.Request(ctx, request)
}

func (t *traceServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Close(ctx, conn)
	}
	return t.concise.Close(ctx, conn)
}
