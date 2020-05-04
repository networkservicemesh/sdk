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

package flags

import (
	"bytes"
	"encoding/csv"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

const (
	urlSliceType = "url-slice"
)

type urlSliceValue struct {
	values  *[]*url.URL
	changed bool
}

// NewURLSliceValue - create new value for URLs
func NewURLSliceValue(p *[]*url.URL) pflag.Value {
	v := &urlSliceValue{
		values:  p,
		changed: false,
	}
	*p = []*url.URL{}
	return v
}

func readAsCSV(val string) ([]*url.URL, error) {
	if val == "" {
		return []*url.URL{}, nil
	}
	stringReader := strings.NewReader(val)
	csvReader := csv.NewReader(stringReader)
	strUrls, err := csvReader.Read()
	if err != nil {
		return nil, errors.Errorf("failed to parse slice of URLs: %q", err)
	}
	result := []*url.URL{}
	for _, urlVal := range strUrls {
		u, err := url.Parse(strings.TrimSpace(urlVal))
		if err != nil || u == nil || u.String() == "" {
			return nil, errors.Errorf("failed to parse URL: %q", urlVal)
		}
		result = append(result, u)
	}
	return result, nil
}

func writeAsCSV(urlVals []*url.URL) (string, error) {
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	vals := []string{}
	for _, v := range urlVals {
		vals = append(vals, v.String())
	}
	err := w.Write(vals)
	if err != nil {
		return "", err
	}
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n"), nil
}

// String - return string value of URL slice
func (s *urlSliceValue) String() string {
	result, err := writeAsCSV(*s.values)
	if err != nil {
		logrus.Errorf("failed to convert to URLs slice %v", err)
		return ""
	}
	return result
}

// Set - update value store, append new values if not default to end of list
func (s *urlSliceValue) Set(val string) error {
	urls, err := readAsCSV(val)
	if err != nil {
		return errors.Errorf("failed to parse URL: %q err: %v", val, err)
	}
	if !s.changed {
		*s.values = urls
		s.changed = true
	} else {
		*s.values = append(*s.values, urls...)
	}
	return nil
}

// Type - return type == url-slice
func (s *urlSliceValue) Type() string {
	return urlSliceType
}

// Append - append new value to list of urls.
func (s *urlSliceValue) Append(val string) error {
	u, err := url.Parse(strings.TrimSpace(val))
	if err != nil || u == nil || u.String() == "" {
		return errors.Errorf("failed to parse URL: %q", val)
	}
	*s.values = append(*s.values, u)
	s.changed = true
	return nil
}

// Replace - replace currently set of values to new ones.
func (s *urlSliceValue) Replace(val []string) error {
	result := []*url.URL{}
	for _, urlVal := range val {
		u, err := url.Parse(strings.TrimSpace(urlVal))
		if err != nil || u == nil || u.String() == "" {
			return errors.Errorf("failed to parse URL: %q", urlVal)
		}
		result = append(result, u)
	}
	*s.values = result
	s.changed = true
	return nil
}

// GetSlice - return values as encoded strings.
func (s *urlSliceValue) GetSlice() []string {
	vals := []string{}
	for _, v := range *s.values {
		vals = append(vals, v.String())
	}
	return vals
}

func getFlagType(f *pflag.FlagSet, name, ftype string, convFunc func(sval string) (interface{}, error)) (interface{}, error) {
	flag := f.Lookup(name)
	if flag == nil {
		err := errors.Errorf("flag accessed but not defined: %s", name)
		return nil, err
	}

	if flag.Value.Type() != ftype {
		err := errors.Errorf("trying to get %s value of flag of type %s", ftype, flag.Value.Type())
		return nil, err
	}

	result, err := convFunc(flag.Value.String())
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetURLSlice return the []*url.URL value of a flag with the given name
func GetURLSlice(f *pflag.FlagSet, name string) ([]*url.URL, error) {
	val, err := getFlagType(f, name, urlSliceType,
		func(sval string) (interface{}, error) {
			return readAsCSV(sval)
		})
	if err != nil {
		return []*url.URL{}, err
	}
	return val.([]*url.URL), nil
}

// URLSliceVarP - defines a url slice flag with specified name, shorthand, default value, and usage string.
func URLSliceVarP(flags *pflag.FlagSet, u *[]*url.URL, name, shorthand string, value []*url.URL, usage string) {
	URLValue := NewURLSliceValue(u)
	*URLValue.(*urlSliceValue).values = value
	flags.VarP(URLValue, name, shorthand, usage)
}
