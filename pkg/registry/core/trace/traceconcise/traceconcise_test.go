// Copyright (c) 2023-2024 Cisco and/or its affiliates.
//
// Copyright (c) 2024 Nordix Foundation.
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

// Package traceconcise_test has few tests for logs in concise mode
package traceconcise_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/registry/core/trace/testutil"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/stretchr/testify/require"
)

func TestOutput(t *testing.T) {
	t.Skip()
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var buff bytes.Buffer
	logrus.SetOutput(&buff)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	log.EnableTracing(true)
	logrus.SetLevel(logrus.InfoLevel)

	s := chain.NewNetworkServiceEndpointRegistryServer(
		memory.NewNetworkServiceEndpointRegistryServer(),
		null.NewNetworkServiceEndpointRegistryServer(),
	)

	nse, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "a",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceEndpointResponse, 1)
	defer close(ch)
	_ = s.Find(&registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch))

	_, err = s.Unregister(context.Background(), nse)
	require.NoError(t, err)

	expectedOutput :=
		" [INFO] [type:registry] nse-server-register={\"name\":\"a\"}\n" +
			" [INFO] [type:registry] nse-server-register-response={\"name\":\"a\"}\n" +
			" [INFO] [type:registry] nse-server-find={\"network_service_endpoint\":{\"name\":\"a\"}}\n" +
			" [INFO] [type:registry] nse-server-send={\"network_service_endpoint\":{\"name\":\"a\"}}\n" +
			" [INFO] [type:registry] nse-server-send-response={\"network_service_endpoint\":{\"name\":\"a\"}}\n" +
			" [INFO] [type:registry] nse-server-find-response={\"network_service_endpoint\":{\"name\":\"a\"}}\n" +
			" [INFO] [type:registry] nse-server-unregister={\"name\":\"a\"}\n" +
			" [INFO] [type:registry] nse-server-unregister-response={\"name\":\"a\"}\n"

	result := testutil.TrimLogTime(&buff)
	require.Equal(t, expectedOutput, result)
}

func TestErrorOutput(t *testing.T) {
	t.Skip()
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var buff bytes.Buffer
	logrus.SetOutput(&buff)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	log.EnableTracing(true)
	logrus.SetLevel(logrus.InfoLevel)

	s := chain.NewNetworkServiceEndpointRegistryServer(
		memory.NewNetworkServiceEndpointRegistryServer(),
		injecterror.NewNetworkServiceEndpointRegistryServer(
			injecterror.WithError(errors.New("test error")),
			injecterror.WithRegisterErrorTimes(0),
			injecterror.WithFindErrorTimes(0),
			injecterror.WithUnregisterErrorTimes(0),
		),
	)

	nse, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "a",
	})
	require.Error(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceEndpointResponse, 1)
	defer close(ch)
	err = s.Find(&registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch))
	require.Error(t, err)

	_, err = s.Unregister(context.Background(), nse)
	require.Error(t, err)

	expectedOutput :=
		" [INFO] [type:registry] nse-server-register={\"name\":\"a\"}\n" +
			" [ERRO] [type:registry] Error returned from sdk/pkg/registry/utils/inject/injecterror/injectErrorNSEServer.Register: test error\n" +
			" [INFO] [type:registry] nse-server-find={\"network_service_endpoint\":{\"name\":\"a\"}}\n" +
			" [ERRO] [type:registry] Error returned from sdk/pkg/registry/utils/inject/injecterror/injectErrorNSEServer.Find: test error\n" +
			" [INFO] [type:registry] nse-server-unregister=null\n" +
			" [ERRO] [type:registry] Error returned from sdk/pkg/registry/utils/inject/injecterror/injectErrorNSEServer.Unregister: test error\n"

	result := testutil.TrimLogTime(&buff)
	require.Equal(t, expectedOutput, result)
}
