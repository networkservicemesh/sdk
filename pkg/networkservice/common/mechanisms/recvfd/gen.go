// Copyright (c) 2020 Cisco and/or its affiliates.
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

package recvfd

import (
	"net/url"
	"os"
	"sync"

	"github.com/edwarnicke/serialize"
)

//go:generate go-syncmap -output per_connection_file_map.gen.go -type perConnectionFileMapMap<string,*perConnectionFileMap>

// PerConnectionFileMap - sync.Map with key == string and value == *perConnectionFileMap
type perConnectionFileMapMap sync.Map

type perConnectionFileMap struct {
	executor           serialize.Executor
	filesByInodeURL    map[string]*os.File
	inodeURLbyFilename map[string]*url.URL
}
