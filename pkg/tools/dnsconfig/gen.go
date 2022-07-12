// Copyright (c) 2022 Cisco Systems, Inc.
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

// Package dnsconfig provides sync map like a Go map[string][]*DNSConfig but is safe for concurrent using
package dnsconfig

import "sync"

//go:generate go-syncmap -output sync_map.gen.go -type Map<string,[]*github.com/networkservicemesh/api/pkg/api/networkservice.DNSConfig>

// Map is like a Go map[string][]*DNSConfig but is safe for concurrent use
// by multiple goroutines without additional locking or coordination
type Map sync.Map
