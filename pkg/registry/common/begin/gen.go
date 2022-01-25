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

package begin

import (
	"sync"
)

//go:generate go-syncmap -output nse_client_map.gen.go -type nseClientMap<string,*eventNSEFactoryClient>
//go:generate go-syncmap -output nse_server_map.gen.go -type nseServerMap<string,*eventNSEFactoryServer>

//go:generate go-syncmap -output ns_client_map.gen.go -type nsClientMap<string,*eventNSFactoryClient>
//go:generate go-syncmap -output ns_server_map.gen.go -type nsServerMap<string,*eventNSFactoryServer>

// nseClientMap - sync.Map with key == string and value == *eventNSEFactoryClient
type nseClientMap sync.Map

// nseServerMap - sync.Map with key == string and value == *eventNSEFactoryClient
type nseServerMap sync.Map

// nsClientMap - sync.Map with key == string and value == *eventNSFactoryClient
type nsClientMap sync.Map

// nsServerMap - sync.Map with key == string and value == *eventNSFactoryClient
type nsServerMap sync.Map
