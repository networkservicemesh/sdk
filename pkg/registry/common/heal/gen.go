// Copyright (c) 2020 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package heal

import (
	"sync"
)

//go:generate go-syncmap -output ctx_wrapper_map.gen.go -type ctxWrapperMap<string,*ctxWrapper>
//go:generate go-syncmap -output find_ctx_wrapper_map.gen.go -type findCtxWrapperMap<string,*findCtxWrapper>

type ctxWrapperMap sync.Map
type findCtxWrapperMap sync.Map

//go:generate go-syncmap -output nse_info_map.gen.go -type nseInfoMap<string,*nseInfo>
//go:generate go-syncmap -output ns_info_map.gen.go -type nsInfoMap<string,*nsInfo>

type nseInfoMap sync.Map
type nsInfoMap sync.Map
