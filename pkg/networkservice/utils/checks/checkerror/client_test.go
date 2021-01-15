// Copyright (c) 2020 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package checkerror_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkerror"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
)

func TestCheckErrorClient_Nil(t *testing.T) {
	request := &networkservice.NetworkServiceRequest{}
	client := chain.NewNetworkServiceClient(
		checkerror.NewClient(t, true),
		null.NewClient(),
	)
	conn, _ := client.Request(context.Background(), request)
	_, err := client.Close(context.Background(), conn)
	assert.Nil(t, err)
}

func TestCheckErrorClient_NotNil(t *testing.T) {
	request := &networkservice.NetworkServiceRequest{}
	client := chain.NewNetworkServiceClient(
		checkerror.NewClient(t, false),
		injecterror.NewClient(),
	)
	conn, _ := client.Request(context.Background(), request)
	_, err := client.Close(context.Background(), conn)
	assert.NotNil(t, err)
}

func TestCheckErrorClient_SpecificError(t *testing.T) {
	request := &networkservice.NetworkServiceRequest{}
	err := errors.New("testerror")
	client := chain.NewNetworkServiceClient(
		checkerror.NewClient(t, false, err),
		injecterror.NewClient(err),
	)
	conn, _ := client.Request(context.Background(), request)
	_, returnedErr := client.Close(context.Background(), conn)
	assert.Equal(t, err, returnedErr)
}
