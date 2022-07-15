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

package opa_test

import (
	"context"
	// "fmt"
	"testing"
	// "time"
	// "github.com/spiffe/go-spiffe/v2/spiffeid"
	// "github.com/spiffe/go-spiffe/v2/svid/x509svid"
	// "github.com/spiffe/go-spiffe/v2/workloadapi"
	// "github.com/stretchr/testify/require"
	// "github.com/networkservicemesh/sdk/pkg/tools/opa"
)

func TestWithServiceConnectionPolicy(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())

	defer cancel()

	// source, err := workloadapi.NewX509Source(ctx)
	// if err != nil {
	// 	fmt.Errorf("couldn't get source %w", err)
	// }
	// fmt.Printf("New Source %v", source)
	// validSvid, err := source.GetX509SVID()
	// if err != nil {
	// 	fmt.Errorf("couldn't get SVID %w", err)
	// }
	// fmt.Printf("New SVID %v", validSvid)

	// id := &spiffeid.ID{td: "", path: "some/path"}
	// invalidSvid := &x509svid.SVID{
	// 	ID: id,
	// }

	// p := opa.WithServiceOwnConnectionPolicy()
	// err := p.Check(ctx, nil)
	// require.Nil(t, err)
	// err = p.Check(context.Background(), invalidSvid)
	// require.NotNil(t, err)
}
