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

package begin

import (
	"sync"
)

//go:generate go run github.com/searKing/golang/tools/cmd/go-syncmap -output client_map.gen.go -type clientMap<string,*eventFactoryClient>

// clientMap - sync.Map with key == string and value == *eventFactoryClient
type clientMap sync.Map

//go:generate go run github.com/searKing/golang/tools/cmd/go-syncmap -output server_map.gen.go -type serverMap<string,*eventFactoryServer>

// serverMap - sync.Map with key == string and value == *eventFactoryServer
type serverMap sync.Map
