// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
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

package dnscontext

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

// Manager can store, remove []dnscontext.Config and also present it as corefile.
// See what is corefile here: https://coredns.io/2017/07/23/corefile-explained/
type Manager struct {
	configs sync.Map
}

func (m *Manager) String() string {
	var keys []string
	result := map[string][]string{}
	conflict := map[string]bool{}
	m.configs.Range(func(_, value interface{}) bool {
		configs := value.([]*networkservice.DNSConfig)
		for _, c := range configs {
			k := strings.Join(c.SearchDomains, " ")
			if len(result[k]) != 0 {
				conflict[k] = true
			} else {
				keys = append(keys, k)
			}
			result[k] = removeDuplicates(append(result[k], c.DnsServerIps...))
		}
		return true
	})
	sort.Strings(keys)
	sb := strings.Builder{}
	i := 0
	for _, k := range keys {
		v := result[k]
		plugin := defaultPlugin
		sort.Strings(v)
		if k == "" {
			_, _ = sb.WriteString(fmt.Sprintf(serverBlockTemplate, AnyDomain, plugin, strings.Join(v, " "), "log\n\treload\n\tcache {\n\t\tdenial 0\n\t}"))
		} else {
			_, _ = sb.WriteString(fmt.Sprintf(serverBlockTemplate, k, plugin, strings.Join(v, " "), "log\n\tcache {\n\t\tdenial 0\n\t}"))
		}
		i++
		if i < len(result) {
			_, _ = sb.WriteRune('\n')
		}
	}
	return sb.String()
}

// Store stores new config with specific id
func (m *Manager) Store(id string, configs ...*networkservice.DNSConfig) {
	m.configs.Store(id, configs)
}

// Remove removes dns config by id
func (m *Manager) Remove(id string) {
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
