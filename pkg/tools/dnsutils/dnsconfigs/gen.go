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

package dnsconfigs

import "sync"

//go:generate go-syncmap -output ip_map.gen.go -type DNSServerIpMap<string,[]net/url.URL>

// DNSServerIpMap is like a Go map[string][]url.URL but is safe for concurrent use
// by multiple goroutines without additional locking or coordination
type DNSServerIpMap sync.Map

//go:generate go-syncmap -output domains_map.gen.go -type SearchDomainsMap<string,[]string>

// SearchDomainsMap is like a Go map[string][]string but is safe for concurrent use
// by multiple goroutines without additional locking or coordination
type SearchDomainsMap sync.Map
