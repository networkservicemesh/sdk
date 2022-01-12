// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

const (
	nameServerKey = "nameserver"
)

// ResolveConfig provides API for editing / reading resolv.conf
// https://man7.org/linux/man-pages/man5/resolv.conf.5.html
type ResolveConfig struct {
	path       string
	properties map[string][]string

	NameServers []string
}

// OpenResolveConfig reads resolve config file from specific path
func OpenResolveConfig(p string) (*ResolveConfig, error) {
	r := &ResolveConfig{
		path:       p,
		properties: make(map[string][]string),
	}
	if err := r.readProperties(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *ResolveConfig) readProperties() error {
	b, err := ioutil.ReadFile(r.path)
	if err != nil {
		return err
	}

	for _, l := range strings.Split(string(b), "\n") {
		words := strings.Split(l, " ")
		switch key := words[0]; key {
		case nameServerKey:
			if len(words) != 2 {
				return errors.Errorf("expected single value for `%s` key, got: %s", nameServerKey, l)
			}
			r.NameServers = append(r.NameServers, words[1])
		default:
			if len(words) > 1 {
				r.properties[key] = words[1:]
			}
		}
	}
	return nil
}

// Value returns value of property
func (r *ResolveConfig) Value(k string) []string {
	return r.properties[k]
}

// SetValue sets value for specific property
func (r *ResolveConfig) SetValue(k string, values ...string) {
	if len(values) == 0 {
		delete(r.properties, k)
	} else {
		r.properties[k] = values
	}
}

// Save saves resolve config file
func (r *ResolveConfig) Save() error {
	var sb strings.Builder
	for _, nameServer := range r.NameServers {
		_, _ = sb.WriteString(nameServerKey)
		_, _ = sb.WriteRune(' ')
		_, _ = sb.WriteString(nameServer)
		_, _ = sb.WriteRune('\n')
	}

	var keys []string
	for k := range r.properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		_, _ = sb.WriteString(k)
		_, _ = sb.WriteRune(' ')
		_, _ = sb.WriteString(strings.Join(r.properties[k], " "))
		_, _ = sb.WriteRune('\n')
	}
	return ioutil.WriteFile(r.path, []byte(sb.String()), os.ModePerm)
}
