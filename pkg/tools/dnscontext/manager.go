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

package dnscontext

import (
	"fmt"
	"sort"
	"strings"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

const (
	anyDomain              = "."
	defaultPlugin          = "forward"
	conflictResolverPlugin = "fanout"
)

// Manager can store, remove []dnscontext.Config and also present it as corefile.
// See what is corefile here: https://coredns.io/2017/07/23/corefile-explained/
type Manager struct {
	configs dnsConfigsMap
}

func (m *Manager) String() string {
	var domains []string
	ipsByDomain := map[string][]string{}
	conflictByDomain := map[string]bool{}
	m.configs.Range(func(_ string, configs []*networkservice.DNSConfig) bool {
		for _, c := range configs {
			domain := strings.Join(c.SearchDomains, " ")
			if domain == "" {
				domain = anyDomain
			}

			if _, ok := ipsByDomain[domain]; ok {
				conflictByDomain[domain] = true
			} else {
				domains = append(domains, domain)
			}

			ipsByDomain[domain] = removeDuplicates(append(ipsByDomain[domain], c.DnsServerIps...))
		}
		return true
	})

	sort.Strings(domains)

	var sb strings.Builder
	for _, domain := range domains {
		plugin := defaultPlugin
		if conflictByDomain[domain] {
			plugin = conflictResolverPlugin
		}

		ips := ipsByDomain[domain]
		sort.Strings(ips)
		ipsString := strings.TrimSpace(strings.Join(ips, " "))

		_, _ = sb.WriteString(fmt.Sprintf("%s {\n", domain))
		if ipsString != "" {
			_, _ = sb.WriteString(fmt.Sprintf("\t%s . %s\n", plugin, ipsString))
		}
		_, _ = sb.WriteString("\tlog\n")
		if domain == anyDomain {
			_, _ = sb.WriteString("\treload\n")
		}
		sb.WriteString("}\n")
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
