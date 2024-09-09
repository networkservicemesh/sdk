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

package awarenessgroups

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AwarenessGroupsDecoder_EmptyInput(t *testing.T) {
	var expected [][]*url.URL
	var decoder Decoder
	err := decoder.Decode(``)
	require.NoError(t, err)
	require.Equal(t, expected, [][]*url.URL(decoder))
}

func Test_AwarenessGroupsDecoder_CorrectInput(t *testing.T) {
	url1, err := url.Parse("kernel://ns-1/nsm-1?color=red")
	require.NoError(t, err)
	url2, err := url.Parse("kernel://ns-2/nsm-2?color=blue")
	require.NoError(t, err)
	url3, err := url.Parse("kernel://ns-3/nsm-3?color=yellow")
	require.NoError(t, err)
	url4, err := url.Parse("kernel://ns-4/nsm-4?color=white")
	require.NoError(t, err)

	expected := [][]*url.URL{
		{url1, url2},
		{url3},
		{url4},
	}
	var decoder Decoder
	err = decoder.Decode(fmt.Sprintf("[%v, %v], [%v], [%v]", url1, url2, url3, url4))
	require.NoError(t, err)
	require.Equal(t, expected, [][]*url.URL(decoder))
}

func Test_AwarenessGroupsDecoder_WrongInput(t *testing.T) {
	var decoder Decoder
	err := decoder.Decode("[a, b], [[c],,[d]")
	require.Error(t, err)

	err = decoder.Decode("[a, b")
	require.Error(t, err)

	err = decoder.Decode("[a, b],")
	require.Error(t, err)

	err = decoder.Decode("[a, b][c]")
	require.Error(t, err)

	err = decoder.Decode("[a, b],[]")
	require.Error(t, err)

	err = decoder.Decode("[a,, b]")
	require.Error(t, err)
}
