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
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

const (
	fmtArg = "--ss=%s"
)

func setUpSSFlagSet(ssp *[]*url.URL) *pflag.FlagSet {
	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	URLSliceVarP(f, ssp, "ss", "s", []*url.URL{}, "Command separated list!")
	return f
}

func setUpSSFlagSetWithDefault(ssp *[]*url.URL) *pflag.FlagSet {
	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	URLSliceVarP(f, ssp, "ss", "s", []*url.URL{
		{Scheme: "unix", Path: "default"}, {Scheme: "unix", Path: "values"}}, "Command separated list!")
	return f
}

func TestEmptySS(t *testing.T) {
	var ss []*url.URL
	f := setUpSSFlagSet(&ss)
	err := f.Parse([]string{})
	if err != nil {
		t.Fatal("expected no error; got", err)
	}
	if len(ss) != 0 {
		t.Fatalf("got ss %v with len=%d but expected length=0", ss, len(ss))
	}
}

func TestEmptySSValue(t *testing.T) {
	var ss []*url.URL
	f := setUpSSFlagSet(&ss)
	err := f.Parse([]string{"--ss="})
	if err != nil {
		t.Fatal("expected no error; got", err)
	}

	getSS, err := GetURLSlice(f, "ss")
	if err != nil {
		t.Fatal("got an error from GetStringSlice():", err)
	}
	if len(getSS) != 0 {
		t.Fatalf("got ss %v with len=%d but expected length=0", getSS, len(getSS))
	}
}

func TestSS(t *testing.T) {
	var ss []*url.URL
	f := setUpSSFlagSet(&ss)

	vals := []string{"unix:///one", "unix:///two", "tcp://127.0.0.1", "tcp://30.30.30.1"}
	arg := fmt.Sprintf("--ss=%s", strings.Join(vals, ","))
	err := f.Parse([]string{arg})
	if err != nil {
		t.Fatal("expected no error; got", err)
	}
	for i, v := range ss {
		if vals[i] != v.String() {
			t.Fatalf("expected ss[%d] to be %s but got: %s", i, vals[i], v)
		}
	}

	getSS, err := GetURLSlice(f, "ss")
	if err != nil {
		t.Fatal("got an error from GetStringSlice():", err)
	}
	for i, v := range getSS {
		if vals[i] != v.String() {
			t.Fatalf("expected ss[%d] to be %s from GetStringSlice but got: %s", i, vals[i], v)
		}
	}
}

func TestSSDefault(t *testing.T) {
	var ss []*url.URL
	f := setUpSSFlagSetWithDefault(&ss)

	vals := []string{"unix://default", "unix://values"}

	err := f.Parse([]string{})
	if err != nil {
		t.Fatal("expected no error; got", err)
	}
	for i, v := range ss {
		if vals[i] != v.String() {
			t.Fatalf("expected ss[%d] to be %s but got: %s", i, vals[i], v)
		}
	}

	getSS, err := GetURLSlice(f, "ss")
	if err != nil {
		t.Fatal("got an error from GetStringSlice():", err)
	}
	for i, v := range getSS {
		if vals[i] != v.String() {
			t.Fatalf("expected ss[%d] to be %s from GetStringSlice but got: %s", i, vals[i], v)
		}
	}
}

func TestSSWithDefault(t *testing.T) {
	var ss []*url.URL
	f := setUpSSFlagSetWithDefault(&ss)

	vals := []string{"unix://one", "unix://two", "tcp://4.4.4.4", "tcp://3.3.3.3"}
	arg := fmt.Sprintf("--ss=%s", strings.Join(vals, ","))
	err := f.Parse([]string{arg})
	if err != nil {
		t.Fatal("expected no error; got", err)
	}
	for i, v := range ss {
		if vals[i] != v.String() {
			t.Fatalf("expected ss[%d] to be %s but got: %s", i, vals[i], v)
		}
	}

	getSS, err := GetURLSlice(f, "ss")
	if err != nil {
		t.Fatal("got an error from GetStringSlice():", err)
	}
	for i, v := range getSS {
		if vals[i] != v.String() {
			t.Fatalf("expected ss[%d] to be %s from GetStringSlice but got: %s", i, vals[i], v)
		}
	}
}

func TestSSCalledTwice(t *testing.T) {
	var ss []*url.URL
	f := setUpSSFlagSet(&ss)

	in := []string{"unix://one,unix://two", "unix://three"}
	expected := []string{"unix://one", "unix://two", "unix://three"}
	arg1 := fmt.Sprintf(fmtArg, in[0])
	arg2 := fmt.Sprintf(fmtArg, in[1])
	err := f.Parse([]string{arg1, arg2})
	if err != nil {
		t.Fatal("expected no error; got", err)
	}

	if len(expected) != len(ss) {
		t.Fatalf("expected number of ss to be %d but got: %d", len(expected), len(ss))
	}
	for i, v := range ss {
		if expected[i] != v.String() {
			t.Fatalf("expected ss[%d] to be %s but got: %s", i, expected[i], v)
		}
	}

	values, err := GetURLSlice(f, "ss")
	if err != nil {
		t.Fatal("expected no error; got", err)
	}

	if len(expected) != len(values) {
		t.Fatalf("expected number of values to be %d but got: %d", len(expected), len(ss))
	}
	for i, v := range values {
		if expected[i] != v.String() {
			t.Fatalf("expected got ss[%d] to be %s but got: %s", i, expected[i], v)
		}
	}
}

func TestSSWithComma(t *testing.T) {
	var ss []*url.URL
	f := setUpSSFlagSet(&ss)

	in := []string{`"unix://one,unix://two"`, `"unix://three"`, `"unix://four,unix://five",unix://six`}
	expected := []string{"unix://one,unix://two", "unix://three", "unix://four,unix://five", "unix://six"}

	arg1 := fmt.Sprintf(fmtArg, in[0])
	arg2 := fmt.Sprintf(fmtArg, in[1])
	arg3 := fmt.Sprintf(fmtArg, in[2])
	err := f.Parse([]string{arg1, arg2, arg3})
	if err != nil {
		t.Fatal("expected no error; got", err)
	}

	if len(expected) != len(ss) {
		t.Fatalf("expected number of ss to be %d but got: %d", len(expected), len(ss))
	}
	for i, v := range ss {
		if expected[i] != v.String() {
			t.Fatalf("expected ss[%d] to be %s but got: %s", i, expected[i], v)
		}
	}

	values, err := GetURLSlice(f, "ss")
	if err != nil {
		t.Fatal("expected no error; got", err)
	}

	if len(expected) != len(values) {
		t.Fatalf("expected number of values to be %d but got: %d", len(expected), len(values))
	}
	for i, v := range values {
		if expected[i] != v.String() {
			t.Fatalf("expected got ss[%d] to be %s but got: %s", i, expected[i], v)
		}
	}
}

func TestSSAsSliceValue(t *testing.T) {
	var ss []*url.URL
	f := setUpSSFlagSet(&ss)

	in := []string{"unix://one", "unix://two"}

	arg1 := fmt.Sprintf(fmtArg, in[0])
	arg2 := fmt.Sprintf(fmtArg, in[1])
	err := f.Parse([]string{arg1, arg2})
	if err != nil {
		t.Fatal("expected no error; got", err)
	}

	f.VisitAll(func(f *pflag.Flag) {
		if val, ok := f.Value.(pflag.SliceValue); ok {
			_ = val.Replace([]string{"unix://three"})
		}
	})
	if len(ss) != 1 || ss[0].String() != "unix://three" {
		t.Fatalf("Expected ss to be overwritten with 'three', but got: %s", ss)
	}
}
