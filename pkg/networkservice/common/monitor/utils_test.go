// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package monitor_test

import (
	"context"
	"net/url"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func startEndpoint(ctx context.Context, listenOn *url.URL, server endpoint.Endpoint) error {
	grpcServer := grpc.NewServer()
	server.Register(grpcServer)

	errCh := grpcutils.ListenAndServe(ctx, listenOn, grpcServer)
	select {
	case err := <-errCh:
		return err
	default:
	}

	return waitNetworkServiceReady(listenOn)
}

func waitNetworkServiceReady(target *url.URL) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(target), grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer func() {
		_ = cc.Close()
	}()

	healthCheckRequest := &grpc_health_v1.HealthCheckRequest{
		Service: networkservice.ServiceNames(null.NewServer())[0],
	}

	client := grpc_health_v1.NewHealthClient(cc)
	for ctx.Err() == nil {
		response, err := client.Check(ctx, healthCheckRequest)
		if err != nil {
			return err
		}
		if response.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			return nil
		}
	}
	return ctx.Err()
}

func waitServerStopped(target *url.URL) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var err error
	for err == nil && ctx.Err() == nil {
		dialCtx, dialCancel := context.WithTimeout(ctx, 10*time.Millisecond)

		var cc *grpc.ClientConn
		if cc, err = grpc.DialContext(dialCtx, grpcutils.URLToTarget(target), grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials())); err == nil {
			_ = cc.Close()
		}

		dialCancel()
	}
	return ctx.Err()
}
