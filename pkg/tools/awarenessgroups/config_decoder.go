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

// Package awarenessgroups provides awareness groups specific tools
package awarenessgroups

import (
	"net/url"
	"strings"
)

// Decoder allows to parse [][]*url.URL from string. Can be used for env configuration.
// See at https://github.com/kelseyhightower/envconfig#custom-decoders
type Decoder [][]*url.URL

// Decode parses values from passed string.
func (d *Decoder) Decode(value string) error {
	value = strings.ReplaceAll(value, " ", "")
	lists := strings.Split(value, "],[")
	awarenessGroups := make([][]*url.URL, len(lists))
	for i, list := range lists {
		list = strings.Trim(list, "[]")
		groupItems := strings.Split(list, ",")
		for _, item := range groupItems {
			nsurl, err := url.Parse(item)
			if err != nil {
				return err
			}
			awarenessGroups[i] = append(awarenessGroups[i], nsurl)
		}
	}

	*d = Decoder(awarenessGroups)
	return nil
}
