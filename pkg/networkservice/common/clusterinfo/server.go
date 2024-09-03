// Copyright (c) 2022 Cisco and/or its affiliates.
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

// Package clusterinfo provides a chain element that appends clusterinfo labels into the request.
package clusterinfo

import (
	"context"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"gopkg.in/yaml.v2"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type clusterinfoServer struct {
	configPath string
}

// NewServer - returns a new clusterinfo NetworkServiceServer that adds clusterinfo labels into request from the cluterinfo configuration.
func NewServer(opts ...Option) networkservice.NetworkServiceServer {
	r := &clusterinfoServer{
		configPath: "/etc/clusterinfo/config.yaml",
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (n *clusterinfoServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetLabels() == nil {
		request.GetConnection().Labels = make(map[string]string)
	}

	m := make(map[string]string)

	if b, err := os.ReadFile(n.configPath); err == nil {
		_ = yaml.Unmarshal(b, &m)
	}

	for k, v := range m {
		request.GetConnection().GetLabels()[k] = v
	}

	return next.Server(ctx).Request(ctx, request)
}

func (n *clusterinfoServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
