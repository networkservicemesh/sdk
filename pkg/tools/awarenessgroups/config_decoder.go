// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

	"github.com/pkg/errors"
)

// Decoder allows to parse [][]*url.URL from string. Can be used for env configuration.
// See at https://github.com/kelseyhightower/envconfig#custom-decoders
type Decoder [][]*url.URL

// Decode parses values from passed string.
func (d *Decoder) Decode(value string) error {
	value = strings.ReplaceAll(value, " ", "")
	if value == "" {
		return nil
	}
	if err := validateParentheses(value); err != nil {
		return err
	}

	lists := strings.Split(value, "],[")
	awarenessGroups := make([][]*url.URL, len(lists))
	for i, list := range lists {
		list = strings.Trim(list, "[]")
		groupItems := strings.Split(list, ",")
		for _, item := range groupItems {
			if item == "" {
				return errors.New("empty nsurl")
			}
			nsurl, err := url.Parse(item)
			if err != nil {
				return errors.Wrapf(err, "failed to parse url %s", item)
			}
			awarenessGroups[i] = append(awarenessGroups[i], nsurl)
		}
	}

	*d = Decoder(awarenessGroups)
	return nil
}

//nolint:nolintlint // TODO: write validation fuction for awarenessGroups values.
func validateParentheses(value string) error {
	i := 0
	length := len(value)
	if length < 2 {
		return errors.New("value is too short")
	}

	parenthesesCounter := 0
	for ; i < length; i++ {
		if value[i] == '[' {
			if parenthesesCounter == 1 {
				return errors.Errorf("unexpected character: %c", value[i])
			}
			parenthesesCounter++
		} else if value[i] == ']' {
			if parenthesesCounter == 0 {
				return errors.Errorf("unexpected character: %c", value[i])
			}
			if i+1 == length {
				return nil
			}
			if i+2 == length {
				return errors.New("unexpected end of value")
			}
			if value[i+1] != ',' {
				return errors.Errorf("unexpected character: %c", value[i+1])
			}
			if value[i+2] != '[' {
				return errors.Errorf("unexpected character: %c", value[i+2])
			}
			parenthesesCounter--
		}
	}

	if parenthesesCounter != 0 {
		return errors.New("parenteses are not balanced")
	}
	return nil
}
