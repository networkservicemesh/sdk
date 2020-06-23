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

// +build callback

package callback_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/tools/callback"
)

type nsmTestImpl struct {
}

func (n nsmTestImpl) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.GetConnection().Labels = map[string]string{"LABEL": "Demo1"}
	// Send Back
	return request.GetConnection(), nil
}

func (n nsmTestImpl) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	return nil, nil
}

type idHolder struct {
	ids chan string
}

func (i *idHolder) ClientConnected(id string) {
	i.ids <- id
}

func (i *idHolder) ClientDisconnected(id string) {
	i.ids <- "-" + id
}

type testCallbackServer struct {
	server           *grpc.Server
	listener         net.Listener
	callbackServer   callback.Server
	sockFile         string
	identityProvider callback.IdentityProvider
	tlsCred          credentials.TransportCredentials
}

const socketMask = 0077

func TempSocketFile() string {
	socketFile, _ := ioutil.TempFile("", "callback.socket")
	_ = socketFile.Close()
	_ = os.Remove(socketFile.Name())
	unix.Umask(socketMask)
	return socketFile.Name()
}

func (s *testCallbackServer) init(t testing.TB) {
	// Construct a Server GRPC server
	ops := []grpc.ServerOption{}
	if s.tlsCred != nil {
		ops = append(ops, grpc.Creds(s.tlsCred))
	}
	s.server = grpc.NewServer(ops...)

	if s.identityProvider == nil {
		s.identityProvider = callback.IdentityByAuthority
	}

	var err error
	s.sockFile = TempSocketFile()
	s.listener, err = net.Listen("unix", s.sockFile)
	require.Nil(t, err)

	// Register callback GRPC endpoint and callback.Server will hold callback clients to be callable.
	s.callbackServer = callback.NewServer(s.identityProvider)
	callback.RegisterCallbackServiceServer(s.server, s.callbackServer)
	go func() {
		_ = s.server.Serve(s.listener)
	}()
}

func (s *testCallbackServer) Stop() {
	_ = s.listener.Close()
	s.server.Stop()
	_ = os.Remove(s.sockFile)
}

func (s *testCallbackServer) waitClient(server *testCallbackServer, client *testCallbackClient, t testing.TB) string {
	ids := &idHolder{
		ids: make(chan string),
	}
	server.callbackServer.AddListener(ids)

	clientID := ""
	select {
	case errValue := <-client.callbackClient.ErrChan():
		require.Errorf(t, errValue, "Unexpected error")
	case <-time.After(1 * time.Second):
		require.Fail(t, "Timeout waiting for client")
	case clientID = <-ids.ids:
		require.Equal(t, "callback:my-client", clientID)
	}
	return clientID
}

type testCallbackClient struct {
	clientGRPC     *grpc.Server
	client         *grpc.ClientConn
	callbackClient callback.Client
	ctx            context.Context
	cancel         context.CancelFunc
	useAuthority   string
	clientCreds    credentials.TransportCredentials
}

func (c *testCallbackClient) init(ctx context.Context, addr string, t testing.TB) {
	nseTest := &nsmTestImpl{}

	c.clientGRPC = grpc.NewServer()
	networkservice.RegisterNetworkServiceServer(c.clientGRPC, nseTest)

	// Connect from Client to Server
	var err error
	ops := []grpc.DialOption{grpc.WithBlock()}
	if c.useAuthority != "" {
		ops = append(ops, grpc.WithAuthority(c.useAuthority))
	}
	if c.clientCreds != nil {
		ops = append(ops, grpc.WithTransportCredentials(c.clientCreds))
	} else {
		ops = append(ops, grpc.WithInsecure())
	}
	c.client, err = grpc.DialContext(ctx, addr, ops...)
	require.Nil(t, err)
	// Construct a callback client.
	c.callbackClient = callback.NewClient(c.client, c.clientGRPC)
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.callbackClient.Serve(c.ctx)
}

func (c *testCallbackClient) Stop() {
	c.cancel()
	c.clientGRPC.Stop()
}

func TestServerClientInvoke(t *testing.T) {
	server := &testCallbackServer{}
	server.init(t)
	defer server.Stop()

	// Client stuff setup, construct a client GRPC server a standard way
	client := &testCallbackClient{
		useAuthority: "my-client", // pass to use client authority
	}
	client.init(context.Background(), "unix:"+server.sockFile, t)
	defer client.Stop()

	clientID := server.waitClient(server, client, t)

	// Do a normal GRPC stuff inside.
	nsmClientGRPC, err3 := grpc.DialContext(context.Background(), clientID, server.callbackServer.WithCallbackDialer(), grpc.WithInsecure(), grpc.WithBlock())
	require.Nil(t, err3)
	defer func() { _ = nsmClientGRPC.Close() }()
	nsmClient := networkservice.NewNetworkServiceClient(nsmClientGRPC)

	resp, err2 := nsmClient.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "qwe",
		},
	})
	require.Nil(t, err2)
	require.NotNil(t, resp)
	require.Equal(t, 1, len(resp.Labels))
	require.Equal(t, "Demo1", resp.Labels["LABEL"])
}
func TestServerClientInvokeNoSecurity(t *testing.T) {
	server := &testCallbackServer{}
	server.init(t)
	defer server.Stop()

	// Client stuff setup, construct a client GRPC server a standard way
	client := &testCallbackClient{
		useAuthority: "my-client", // pass to use client authority
	}
	client.init(context.Background(), "unix:"+server.sockFile, t)
	defer client.Stop()

	clientID := server.waitClient(server, client, t)

	// Do a normal GRPC stuff inside.
	nsmClientGRPC, err3 := grpc.DialContext(context.Background(), clientID, server.callbackServer.WithCallbackDialer())
	require.NotNil(t, err3)
	require.Nil(t, nsmClientGRPC)
}

func TestServerClientInvokeClose(t *testing.T) {
	server := &testCallbackServer{}
	server.init(t)
	defer server.Stop()

	// Client stuff setup, construct a client GRPC server a standard way
	client := &testCallbackClient{
		useAuthority: "my-client", // pass to use client authority
	}
	client.init(context.Background(), "unix:"+server.sockFile, t)
	defer client.Stop()

	clientID := server.waitClient(server, client, t)

	// Do a normal GRPC stuff inside.
	nsmClientGRPC, err3 := grpc.DialContext(context.Background(), clientID, server.callbackServer.WithCallbackDialer(), grpc.WithInsecure())
	require.NotNil(t, nsmClientGRPC)
	require.Nil(t, err3)
	err4 := nsmClientGRPC.Close()
	require.Nil(t, err4)
}

// IdentityByAuthority - return identity by :authority
func IdentityByPeerCertificate(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		err := errors.New("No peer is provided")
		logrus.Error(err)
		return "", err
	}
	tlsInfo, tlsOk := p.AuthInfo.(credentials.TLSInfo)
	if !tlsOk {
		err := errors.New("No TLS info is provided")
		logrus.Error(err)
		return "", err
	}
	commonName := tlsInfo.State.PeerCertificates[0].Subject.CommonName
	return commonName, nil
}

func TestServerClientInvokeWithPeerAuth(t *testing.T) {
	serverCrt, serverKey, _ := createPem("my-server")
	serverCert, serr := tls.X509KeyPair(serverCrt, serverKey)
	require.Nil(t, serr)

	clientCrt, clientKey, clientDer := createPem("my-client")
	clientCert, cerr := tls.X509KeyPair(clientCrt, clientKey)
	require.Nil(t, cerr)

	clientPool := x509.NewCertPool()
	clientCrt509, crterr := x509.ParseCertificate(clientDer)
	require.Nil(t, crterr)
	clientPool.AddCert(clientCrt509)
	// Create the TLS credentials
	tlsCred := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAnyClientCert,
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    clientPool,
	})

	server := &testCallbackServer{
		identityProvider: IdentityByPeerCertificate,
		tlsCred:          tlsCred,
	}
	server.init(t)
	defer server.Stop()

	serverName := "unix:" + server.sockFile
	clientCreds := credentials.NewTLS(&tls.Config{
		ServerName:         serverName,
		Certificates:       []tls.Certificate{clientCert},
		RootCAs:            clientPool,
		InsecureSkipVerify: true, //nolint:gosec
	})
	// Client stuff setup, construct a client GRPC server a standard way
	client := &testCallbackClient{
		clientCreds: clientCreds,
	}

	client.init(context.Background(), serverName, t)
	defer client.Stop()

	clientID := server.waitClient(server, client, t)

	// Do a normal GRPC stuff inside.
	nsmClientGRPC, err3 := grpc.DialContext(context.Background(), clientID, server.callbackServer.WithCallbackDialer(), grpc.WithInsecure())
	defer func() { _ = nsmClientGRPC.Close() }()
	require.Nil(t, err3)
	nsmClient := networkservice.NewNetworkServiceClient(nsmClientGRPC)

	resp, err2 := nsmClient.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "qwe",
		},
	})
	require.Nil(t, err2)
	require.NotNil(t, resp)
	require.Equal(t, 1, len(resp.Labels))
	require.Equal(t, "Demo1", resp.Labels["LABEL"])
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func createPem(commonName string) (certpem, keypem, derbytes []byte) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	notBefore := time.Now()
	notAfter := notBefore.AddDate(1, 0, 0)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"NSM Co"},
			CommonName:   commonName,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	template.IsCA = true
	derbytes, _ = x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	certpem = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derbytes})
	keypem = pem.EncodeToMemory(pemBlockForKey(priv))
	return certpem, keypem, derbytes
}

func BenchmarkServerClientInvoke(b *testing.B) {
	server := &testCallbackServer{}
	server.init(b)
	defer server.Stop()

	// Client stuff setup, construct a client GRPC server a standard way
	client := &testCallbackClient{
		useAuthority: "my-client",
	}
	client.init(context.Background(), "unix:"+server.sockFile, b)
	defer client.Stop()

	clientID := server.waitClient(server, client, b)

	// Do a normal GRPC stuff inside.
	nsmClientGRPC, err3 := grpc.DialContext(context.Background(), clientID, server.callbackServer.WithCallbackDialer(), grpc.WithInsecure())
	require.Nil(b, err3)
	nsmClient := networkservice.NewNetworkServiceClient(nsmClientGRPC)

	resp, err2 := nsmClient.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "qwe",
		},
	})
	require.Nil(b, err2)
	require.NotNil(b, resp)
	require.Equal(b, 1, len(resp.Labels))
	require.Equal(b, "Demo1", resp.Labels["LABEL"])

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err2 := nsmClient.Request(context.Background(), &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id: "qwe",
			},
		})
		require.Nil(b, err2)
		require.NotNil(b, resp)
		require.Equal(b, 1, len(resp.Labels))
		require.Equal(b, "Demo1", resp.Labels["LABEL"])
	}
}

func BenchmarkServerClientInvokePure(b *testing.B) {
	// Construct a Server GRPC server
	server := grpc.NewServer()
	sockFile := TempSocketFile()
	listener, err := net.Listen("unix", sockFile)
	defer func() { _ = os.Remove(sockFile) }()
	require.Nil(b, err)
	defer func() { _ = listener.Close() }()
	defer server.Stop()

	// Register callback GRPC endpoint and callback.Server will hold callback clients to be callable.
	// Client stuff setup, construct a client GRPC server a standard way
	nseTest := &nsmTestImpl{}
	networkservice.RegisterNetworkServiceServer(server, nseTest)

	go func() {
		_ = server.Serve(listener)
	}()

	// Connect from Client to Server
	var client *grpc.ClientConn
	client, err = grpc.DialContext(context.Background(), "unix:"+sockFile, grpc.WithInsecure(), grpc.WithAuthority("my-client"), grpc.WithBlock())
	require.Nil(b, err)
	nsmClient := networkservice.NewNetworkServiceClient(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err2 := nsmClient.Request(context.Background(), &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id: "qwe",
			},
		})
		require.Nil(b, err2)
		require.NotNil(b, resp)
		require.Equal(b, 1, len(resp.Labels))
		require.Equal(b, "Demo1", resp.Labels["LABEL"])
	}
}

type nsmTestMonImpl struct {
}

func (n nsmTestMonImpl) MonitorConnections(scope *networkservice.MonitorScopeSelector, server networkservice.MonitorConnection_MonitorConnectionsServer) error {
	logrus.Infof("Begin monitor connections: %v", scope)
	for i := 0; i < 50; i++ {
		err := server.Send(&networkservice.ConnectionEvent{
			Type: 0,
			Connections: map[string]*networkservice.Connection{
				fmt.Sprintf("id-%v", i): {
					Id:    fmt.Sprintf("id-%v", i),
					State: networkservice.State_DOWN,
				},
			},
		})
		if err != nil {
			logrus.Errorf("Err %v", err)
		}
	}
	logrus.Infof("End event send: %v. Wait for server close", scope)
	<-server.Context().Done()
	logrus.Infof("End monitor connections: %v.", scope)
	return nil
}

func TestServerClientConnectionMonitor(t *testing.T) {
	// Construct a Server GRPC server
	server := grpc.NewServer()
	listener, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer func() { _ = listener.Close() }()
	defer server.Stop()
	// Register callback GRPC endpoint and callback.Server will hold callback clients to be callable.
	callbackServer := callback.NewServer(callback.IdentityByAuthority)
	callback.RegisterCallbackServiceServer(server, callbackServer)
	go func() {
		_ = server.Serve(listener)
	}()
	// Client stuff setup, construct a client GRPC server a standard way
	monTest := &nsmTestMonImpl{}

	clientGRPC := grpc.NewServer()
	defer clientGRPC.Stop()
	networkservice.RegisterMonitorConnectionServer(clientGRPC, monTest)

	// Connect from Client to Server
	var client *grpc.ClientConn
	client, err = grpc.DialContext(context.Background(), listener.Addr().String(), grpc.WithInsecure(), grpc.WithAuthority("my-client"), grpc.WithBlock())
	require.Nil(t, err)
	// Construct a callback client.
	callbackClient := callback.NewClient(client, clientGRPC)
	callbackClient.Serve(context.Background())
	ids := &idHolder{
		ids: make(chan string),
	}
	callbackServer.AddListener(ids)
	clientID := waitClientConnected(callbackClient, t, ids)
	nsmClientGRPC, err3 := grpc.DialContext(context.Background(), clientID, callbackServer.WithCallbackDialer(), grpc.WithInsecure())
	defer func() { _ = nsmClientGRPC.Close() }()
	require.Nil(t, err3)
	nsmClient := networkservice.NewMonitorConnectionClient(nsmClientGRPC)
	monCtx, cancelMon := context.WithCancel(context.Background())
	monClient, err2 := nsmClient.MonitorConnections(monCtx, createScopeSelector())
	require.Nil(t, err2)
	events := []*networkservice.ConnectionEvent{}
	logrus.Infof("----------")
	for {
		conEvent, err4 := monClient.Recv()
		if err4 == nil {
			events = append(events, conEvent)
		} else {
			require.Equal(t, err4.Error(), "msg")
			break
		}
		if len(events) == 50 {
			break
		}
	}
	require.Equal(t, 50, len(events))
	cancelMon()
}

func waitClientConnected(callbackClient callback.Client, t *testing.T, ids *idHolder) string {
	clientID := ""
	select {
	case errValue := <-callbackClient.ErrChan():
		require.Errorf(t, errValue, "Unexpected error")
	case <-time.After(1 * time.Second):
		require.Fail(t, "Timeout waiting for client")
	case clientID = <-ids.ids:
		require.Equal(t, "callback:my-client", clientID)
	}
	return clientID
}

func createScopeSelector() *networkservice.MonitorScopeSelector {
	return &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{
			{
				Name:  "segm",
				Token: "tok",
			},
		},
	}
}
