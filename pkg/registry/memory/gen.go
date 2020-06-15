// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package memory

import "sync"

//go:generate go-syncmap -output ns_sync_map.gen.go -type NetworkServiceSyncMap<string,*github.com/networkservicemesh/api/pkg/api/registry.NetworkService>
//go:generate go-syncmap -output nse_sync_map.gen.go -type NetworkServiceEndpointSyncMap<string,*github.com/networkservicemesh/api/pkg/api/registry.NetworkServiceEndpoint>

// NetworkServiceSyncMap is like a Go map[string]*registry.NetworkService but is safe for concurrent use
// by multiple goroutines without additional locking or coordination.
type NetworkServiceSyncMap sync.Map

// NetworkServiceEndpointSyncMap is like a Go map[string]*registry.NetworkServiceEndpoint but is safe for concurrent use
// by multiple goroutines without additional locking or coordination.
type NetworkServiceEndpointSyncMap sync.Map
