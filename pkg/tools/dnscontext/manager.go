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

package dnscontext

import (
	"fmt"
	"strings"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

// Manager provides API for storing/deleting dnsConfigs. Can represent the configs in caddyfile format.
// Can be used from different goroutines
type Manager interface {
	Remove(string)
	Store(string, ...*networkservice.DNSConfig)
	fmt.Stringer
}

type manager struct {
	configs sync.Map
}

func (m *manager) String() string {
	result := map[string][]string{}
	conflict := map[string]bool{}
	m.configs.Range(func(_, value interface{}) bool {
		configs := value.([]*networkservice.DNSConfig)
		for _, c := range configs {
			k := strings.Join(c.SearchDomains, " ")
			if len(result[k]) != 0 {
				conflict[k] = true
			}
			result[k] = removeDuplicates(append(result[k], c.DnsServerIps...))
		}
		return true
	})
	sb := strings.Builder{}
	i := 0
	for k, v := range result {
		plugin := defaultPlugin
		if conflict[k] {
			plugin = conflictResolverPlugin
		}
		if k == "" {
			_, _ = sb.WriteString(fmt.Sprintf(serverBlockTemplate, AnyDomain, plugin, strings.Join(v, " "), "log\n\treload"))
		} else {
			_, _ = sb.WriteString(fmt.Sprintf(serverBlockTemplate, k, plugin, strings.Join(v, " "), "log"))
		}
		i++
		if i < len(result) {
			_, _ = sb.WriteRune('\n')
		}
	}
	return sb.String()
}

// NewManager creates new config manager
func NewManager() Manager {
	return &manager{}
}

// Store stores new config with specific id
func (m *manager) Store(id string, configs ...*networkservice.DNSConfig) {
	m.configs.Store(id, configs)
}

// Remove removes dns config by id
func (m *manager) Remove(id string) {
	m.configs.Delete(id)
}

func removeDuplicates(words []string) []string {
	set := make(map[string]bool)
	var result []string
	for i := 0; i < len(words); i++ {
		if set[words[i]] {
			continue
		}
		set[words[i]] = true
		result = append(result, words[i])
	}
	return result
}
