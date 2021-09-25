// Copyright (c) 2021 Cisco and/or its affiliates.
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

package dial

import (
	"context"
	"net/url"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

type dialInfo struct {
	url *url.URL
	*grpc.ClientConn
}

func (d *dialInfo) Close() error {
	if d != nil && d.ClientConn != nil {
		return d.ClientConn.Close()
	}
	return nil
}

func store(ctx context.Context, dialInfo *dialInfo) {
	metadata.Map(ctx, true).Store(key{}, dialInfo)
}

func loadAndDelete(ctx context.Context) (value *dialInfo, ok bool) {
	m := metadata.Map(ctx, true)
	rawValue, ok := m.LoadAndDelete(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(*dialInfo)
	return value, ok
}
