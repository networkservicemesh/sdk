// Copyright (c) 2023 Cisco Systems, Inc.
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

package netsvcmonitor

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type cancelFunctionKey struct{}

func storeCancelFunction(ctx context.Context, cancel func()) {
	metadata.Map(ctx, false).Store(cancelFunctionKey{}, cancel)
}

func loadCancelFunction(ctx context.Context) (func(), bool) {
	v, ok := metadata.Map(ctx, false).Load(cancelFunctionKey{})
	if ok {
		return v.(func()), true
	}
	return nil, false
}
