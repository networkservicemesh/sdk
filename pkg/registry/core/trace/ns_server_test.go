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
package trace_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace/testutil"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/stretchr/testify/require"
)

func TestOutput_RegistryNetworkService(t *testing.T) {
	t.Skip()
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var buff bytes.Buffer
	logrus.SetOutput(&buff)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	log.EnableTracing(true)
	logrus.SetLevel(logrus.InfoLevel)

	s := chain.NewNetworkServiceRegistryServer(
		memory.NewNetworkServiceRegistryServer(),
		null.NewNetworkServiceRegistryServer(),
	)

	ns, err := s.Register(context.Background(), &registry.NetworkService{
		Name: "a",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceResponse, 1)
	defer close(ch)
	_ = s.Find(&registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceFindServer(ctx, ch))

	_, err = s.Unregister(context.Background(), ns)
	require.NoError(t, err)

	expectedOutput := ` [INFO] [type:registry] ns-server-register=name:"a"
 [INFO] [type:registry] ns-server-register-response=name:"a"
 [INFO] [type:registry] ns-server-find=network_service:{name:"a"}
 [INFO] [type:registry] ns-server-send=network_service:{name:"a"}
 [INFO] [type:registry] ns-server-send-response=network_service:{name:"a"}
 [INFO] [type:registry] ns-server-find-response=network_service:{name:"a"}
 [INFO] [type:registry] ns-server-unregister=name:"a"
 [INFO] [type:registry] ns-server-unregister-response=name:"a"
`

	result := testutil.TrimLogTime(&buff)

	result = testutil.Normalize(result)
	expectedOutput = testutil.Normalize(expectedOutput)

	require.Equal(t, expectedOutput, result)
}

func TestOutput_Trace_RegistryNetworkService(t *testing.T) {
	t.Skip()
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var buff bytes.Buffer
	logrus.SetOutput(&buff)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	log.EnableTracing(true)
	logrus.SetLevel(logrus.TraceLevel)

	s := chain.NewNetworkServiceRegistryServer(
		memory.NewNetworkServiceRegistryServer(),
		null.NewNetworkServiceRegistryServer(),
	)

	ns, err := s.Register(context.Background(), &registry.NetworkService{
		Name: "a",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceResponse, 1)
	defer close(ch)
	_ = s.Find(&registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceFindServer(ctx, ch))

	_, err = s.Unregister(context.Background(), ns)
	require.NoError(t, err)

	expectedOutput := ` [TRAC] [type:registry] (1) ⎆ memory/memoryNSServer.Register()
 [TRAC] [type:registry] (1.1)   register=name:"a"
 [TRAC] [type:registry] (2)  ⎆ null/nullNSServer.Register()
 [TRAC] [type:registry] (2.1)    register=name:"a"
 [TRAC] [type:registry] (2.2)    register-response=name:"a"
 [TRAC] [type:registry] (1.2)   register-response=name:"a"
 [TRAC] [type:registry] (1) ⎆ memory/memoryNSServer.Find()
 [TRAC] [type:registry] (1.1)   find=network_service:{name:"a"}
 [TRAC] [type:registry] (2)  ⎆ memory/memoryNSServer.Send()
 [TRAC] [type:registry] (2.1)    network service=name:"a"
 [TRAC] [type:registry] (2)  ⎆ null/nullNSServer.Find()
 [TRAC] [type:registry] (2.1)    find=network_service:{name:"a"}
 [TRAC] [type:registry] (1) ⎆ memory/memoryNSServer.Unregister()
 [TRAC] [type:registry] (1.1)   unregister=name:"a"
 [TRAC] [type:registry] (2)  ⎆ null/nullNSServer.Unregister()
 [TRAC] [type:registry] (2.1)    unregister=name:"a"
 [TRAC] [type:registry] (2.2)    unregister-response=name:"a"
 [TRAC] [type:registry] (1.2)   unregister-response=name:"a"
`

	result := testutil.TrimLogTime(&buff)
	result = testutil.Normalize(result)
	expectedOutput = testutil.Normalize(expectedOutput)

	require.Equal(t, expectedOutput, result)
}

func TestErrorOutput_RegistryNetworkService(t *testing.T) {
	t.Skip()
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var buff bytes.Buffer
	logrus.SetOutput(&buff)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	log.EnableTracing(true)
	logrus.SetLevel(logrus.InfoLevel)

	s := chain.NewNetworkServiceRegistryServer(
		memory.NewNetworkServiceRegistryServer(),
		injecterror.NewNetworkServiceRegistryServer(
			injecterror.WithError(errors.New("test error")),
			injecterror.WithRegisterErrorTimes(0),
			injecterror.WithFindErrorTimes(0),
			injecterror.WithUnregisterErrorTimes(0),
		),
	)

	ns, err := s.Register(context.Background(), &registry.NetworkService{
		Name: "a",
	})
	require.Error(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceResponse, 1)
	defer close(ch)
	err = s.Find(&registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceFindServer(ctx, ch))
	require.Error(t, err)

	_, err = s.Unregister(context.Background(), ns)
	require.Error(t, err)

	expectedOutput := ` [INFO] [type:registry] ns-server-register=name:"a"
 [ERRO] [type:registry] Error returned from injecterror/injectErrorNSServer.Register: test error
 [INFO] [type:registry] ns-server-find=network_service:{name:"a"}
 [ERRO] [type:registry] Error returned from injecterror/injectErrorNSServer.Find: test error
 [INFO] [type:registry] ns-server-unregister=<nil>
 [ERRO] [type:registry] Error returned from injecterror/injectErrorNSServer.Unregister: test error
`
	result := testutil.TrimLogTime(&buff)

	result = testutil.Normalize(result)
	expectedOutput = testutil.Normalize(expectedOutput)

	require.Equal(t, expectedOutput, result)
}

func TestErrorOutput_Trace_RegistryNetworkService(t *testing.T) {
	t.Skip()
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var buff bytes.Buffer
	logrus.SetOutput(&buff)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	log.EnableTracing(true)
	logrus.SetLevel(logrus.TraceLevel)

	s := chain.NewNetworkServiceRegistryServer(
		memory.NewNetworkServiceRegistryServer(),
		injecterror.NewNetworkServiceRegistryServer(
			injecterror.WithError(&testutil.CustomError{}),
			injecterror.WithRegisterErrorTimes(0),
			injecterror.WithFindErrorTimes(0),
			injecterror.WithUnregisterErrorTimes(0),
		),
	)

	ns, err := s.Register(context.Background(), &registry.NetworkService{
		Name: "a",
	})
	require.Error(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceResponse, 1)
	defer close(ch)
	err = s.Find(&registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceFindServer(ctx, ch))
	require.Error(t, err)

	_, err = s.Unregister(context.Background(), ns)
	require.Error(t, err)

	expectedOutput := ` [TRAC] [type:registry] (1) ⎆ memory/memoryNSServer.Register()
 [TRAC] [type:registry] (1.1)   register=name:"a"
 [TRAC] [type:registry] (2)  ⎆ injecterror/injectErrorNSServer.Register()
 [TRAC] [type:registry] (2.1)    register=name:"a"
 [ERRO] [type:registry] (2.2)    Error returned from api/pkg/api/networkservice/networkServiceClient.Close;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*beginTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:85;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*endTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:106;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	
 [ERRO] [type:registry] (1.2)   Error returned from api/pkg/api/networkservice/networkServiceClient.Close;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*beginTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:85;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*endTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:106;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	
 [TRAC] [type:registry] (1) ⎆ memory/memoryNSServer.Find()
 [TRAC] [type:registry] (1.1)   find=network_service:{name:"a"}
 [TRAC] [type:registry] (2)  ⎆ injecterror/injectErrorNSServer.Find()
 [TRAC] [type:registry] (2.1)    find=network_service:{name:"a"}
 [ERRO] [type:registry] (2.2)    Error returned from api/pkg/api/networkservice/networkServiceClient.Close;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*beginTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:85;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*endTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:106;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	
 [ERRO] [type:registry] (1.2)   Error returned from api/pkg/api/networkservice/networkServiceClient.Close;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*beginTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:85;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*endTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:106;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	
 [TRAC] [type:registry] (1) ⎆ memory/memoryNSServer.Unregister()
 [TRAC] [type:registry] (1.1)   unregister=<nil>
 [TRAC] [type:registry] (2)  ⎆ injecterror/injectErrorNSServer.Unregister()
 [TRAC] [type:registry] (2.1)    unregister=<nil>
 [ERRO] [type:registry] (2.2)    Error returned from api/pkg/api/networkservice/networkServiceClient.Close;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*beginTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:85;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*endTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:106;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	
 [ERRO] [type:registry] (1.2)   Error returned from api/pkg/api/networkservice/networkServiceClient.Close;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*beginTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:85;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*endTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:106;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	
`
	result := testutil.TrimLogTime(&buff)

	result = testutil.Normalize(result)
	expectedOutput = testutil.Normalize(expectedOutput)

	require.Equal(t, expectedOutput, result)
}
