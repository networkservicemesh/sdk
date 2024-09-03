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

package replacensename

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type keyType struct{}

// store - stores the next network service endpoint name.
func store(ctx context.Context, nseName string) {
	metadata.Map(ctx, true).Store(keyType{}, nseName)
}

// load - returns the next network service endpoint name.
func load(ctx context.Context) (value string, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).Load(keyType{})
	if !ok {
		return
	}
	value, ok = rawValue.(string)
	return value, ok
}

// loadAndDelete - returns the next network service endpoint name and deletes it from the metadata.
func loadAndDelete(ctx context.Context) (value string, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).LoadAndDelete(keyType{})
	if !ok {
		return
	}
	value, ok = rawValue.(string)
	return value, ok
}
