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

package expire

import "sync"

//go:generate go-syncmap -output timer_sync_map.gen.go -type timerMap<string,*time.Timer>
//go:generate go-syncmap -output int_sync_map.gen.go -type intMap<string,*int32>
//go:generate go-syncmap -output context_sync_map.gen.go -type contextMap<string,context.Context>
//go:generate go-syncmap -output ns_state_sync_map.gen.go -type nsStateMap<string,*nsState>

type timerMap sync.Map
type contextMap sync.Map
type intMap sync.Map
type nsStateMap sync.Map
