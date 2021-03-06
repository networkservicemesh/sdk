// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package connect

import "sync"

//go:generate go-syncmap -output nse_info_map.gen.go -type nseInfoMap<string,*nseInfo>
//go:generate go-syncmap -output nse_client_map.gen.go -type nseClientMap<string,*nseClient>

type nseInfoMap sync.Map
type nseClientMap sync.Map

//go:generate go-syncmap -output ns_info_map.gen.go -type nsInfoMap<string,*nsInfo>
//go:generate go-syncmap -output ns_client_map.gen.go -type nsClientMap<string,*nsClient>

type nsInfoMap sync.Map
type nsClientMap sync.Map
