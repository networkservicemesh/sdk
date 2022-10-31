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

package grpcmetadata

import (
	"sync"
)

//go:generate go-syncmap -output sync_map.gen.go -type resourcePathMap<string,*github.com/networkservicemesh/api/pkg/api/registry.Path>

// resourcePathMap - sync.Map with key == rosource name and value == path associated with this resource
type resourcePathMap sync.Map
