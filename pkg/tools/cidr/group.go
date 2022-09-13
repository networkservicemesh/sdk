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

package cidr

import (
	"net"
	"strings"

	"github.com/pkg/errors"
)

// Groups allows parsing cidr groups.
// Example: [v1, v2, v3], [v4], v5 =>
//
//	 [][]*net.IPNet{
//			{v1, v2, v3},
//			{v4},
//			{v5},
//		}
//
// Can be used for env configuration.
// See at https://github.com/kelseyhightower/envconfig#custom-decoders
type Groups [][]*net.IPNet

const (
	// inside - means that we are inside a group
	inside int = iota

	// outside - means that we are outside a group
	outside
)

// Decode parses values from passed string.
func (g *Groups) Decode(value string) error {
	value = strings.ReplaceAll(value, " ", "")
	if value == "" {
		return nil
	}
	if err := validateParentheses(value); err != nil {
		return err
	}

	list := strings.Split(value, ",")
	var groups [][]*net.IPNet

	state := outside

	for _, item := range list {
		if state == outside {
			groups = append(groups, []*net.IPNet{})
		}
		toAdd := strings.Trim(item, "[]")
		if toAdd == "" {
			return errors.New("empty value")
		}
		_, ipNet, parseErr := net.ParseCIDR(strings.TrimSpace(toAdd))
		if parseErr != nil {
			return errors.Errorf("сould not parse CIDR %s; %+v", item, parseErr)
		}

		groups[len(groups)-1] = append(groups[len(groups)-1], ipNet)

		if strings.HasSuffix(item, "]") {
			state = outside
		} else if strings.HasPrefix(item, "[") {
			state = inside
		}
	}
	*g = groups
	return nil
}

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
			parenthesesCounter--
		}
	}

	if parenthesesCounter != 0 {
		return errors.New("parenteses are not balanced")
	}
	return nil
}
