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

package flags

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type uRLValue url.URL

// NewURLValue - create new value for URLs
func NewURLValue(p *url.URL) pflag.Value {
	*p = url.URL{}
	return (*uRLValue)(p)
}

func (uv *uRLValue) String() string {
	inner := url.URL(*uv)
	innerPtr := &inner
	return innerPtr.String()
}

func (uv *uRLValue) Set(s string) error {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil || u == nil || u.String() == "" {
		return errors.Errorf("failed to parse URL: %q", s)
	}
	*uv = uRLValue(*u)
	return nil
}

func (uv *uRLValue) Type() string {
	return "url"
}

// URLVarP - defines a url flag with specified name, shorthand, default value, and usage string.
func URLVarP(flags *pflag.FlagSet, u *url.URL, name, shorthand string, value *url.URL, usage string) {
	URLValue := NewURLValue(u)
	_ = URLValue.Set(value.String())
	flags.VarP(URLValue, name, shorthand, usage)
}
