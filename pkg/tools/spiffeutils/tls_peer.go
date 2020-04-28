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

package spiffeutils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spiffe/go-spiffe/spiffe"
)

type selfSignedTLSPeer struct {
	spiffeID      *url.URL
	duration      time.Duration
	renewDuration time.Duration
	renewTimer    *time.Timer
	tlsCert       *tls.Certificate
	x509Cert      *x509.Certificate
	sync.RWMutex
	err error
}

// TLSPeer partial interface to allow substituting alternatives to spiffe.TLSPeer (which is sadly, a struct)
type TLSPeer interface {
	GetConfig(ctx context.Context, expectPeer spiffe.ExpectPeerFunc) (*tls.Config, error)
	GetCertificate() (*tls.Certificate, error)
	Close() error
	WaitUntilReady(ctx context.Context) error
}

type tlsPeerWrapper struct {
	TLSPeer
	selfSignedTLSPeer TLSPeer
	sync.Once
}

func (t *tlsPeerWrapper) replacePeer() {
	t.Do(func() {
		if t.selfSignedTLSPeer != nil {
			old := t.TLSPeer
			t.TLSPeer = t.selfSignedTLSPeer
			if old != nil {
				_ = old.Close()
			}
		}
	})
}

func (t *tlsPeerWrapper) GetConfig(ctx context.Context, expectPeer spiffe.ExpectPeerFunc) (*tls.Config, error) {
	tlsConfig, err := t.TLSPeer.GetConfig(ctx, expectPeer)
	if err != nil {
		t.replacePeer()
		return t.TLSPeer.GetConfig(ctx, expectPeer)
	}
	return tlsConfig, nil
}

func (t *tlsPeerWrapper) GetCertificate() (*tls.Certificate, error) {
	cert, err := t.TLSPeer.GetCertificate()
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func (t *tlsPeerWrapper) WaitUntilReady(ctx context.Context) error {
	err := t.TLSPeer.WaitUntilReady(ctx)
	if err != nil {
		t.replacePeer()
		return t.TLSPeer.WaitUntilReady(ctx)
	}
	return nil
}

func (t *tlsPeerWrapper) newSelfSignedTLSPeer() (TLSPeer, error) {
	spiffeID, err := url.Parse(fmt.Sprintf("spiffe://selfsigned/%s", uuid.New().String()))
	if err != nil {
		return nil, err
	}
	peer, err := NewSelfSignedTLSPeer(spiffeID, 24*time.Hour, 12*time.Hour)
	if err != nil {
		return nil, err
	}
	return peer, nil
}

// NewTLSPeer - returns a new TLSPeer that is either will be backed either by the spiffe.TLSPeer *or* a self signed TLSPeer if the spiffe agent does not respond
func NewTLSPeer(opts ...spiffe.TLSPeerOption) (TLSPeer, error) {
	rv := &tlsPeerWrapper{}
	if ss, err := rv.newSelfSignedTLSPeer(); err == nil {
		rv.selfSignedTLSPeer = ss
	}
	peer, err := spiffe.NewTLSPeer(opts...)
	if err != nil {
		if rv.selfSignedTLSPeer != nil {
			rv.replacePeer()
			return rv, nil
		}
	}
	rv.TLSPeer = peer
	return rv, nil
}

// NewSelfSignedTLSPeer - creates a new TLSPeer with a self signed cert for the provides spiffeID good for duration and rotated every renewDuration
func NewSelfSignedTLSPeer(spiffeID *url.URL, duration, renewDuration time.Duration) (TLSPeer, error) {
	s := &selfSignedTLSPeer{
		spiffeID:      spiffeID,
		duration:      duration,
		renewDuration: renewDuration,
	}
	s.rotateCert()
	if s.err != nil {
		return nil, s.err
	}
	return s, nil
}

func (s *selfSignedTLSPeer) rotateCert() {
	s.Lock()
	defer s.Unlock()
	notBefore := time.Now()
	notAfter := notBefore.Add(s.duration)
	x509cert, priv, err := SelfSignedX509SVID(s.spiffeID, notBefore, notAfter)
	if err != nil {
		s.err = err
		return
	}
	s.x509Cert = x509cert
	s.tlsCert = &tls.Certificate{
		Certificate: [][]byte{x509cert.Raw},
		PrivateKey:  priv,
	}
	s.renewTimer = time.AfterFunc(s.renewDuration, s.rotateCert)
}

func (s *selfSignedTLSPeer) GetCertificate() (*tls.Certificate, error) {
	s.RLock()
	defer s.RUnlock()
	if s.err != nil {
		return nil, s.err
	}
	return s.tlsCert, nil
}

func (s *selfSignedTLSPeer) GetConfig(ctx context.Context, expectPeer spiffe.ExpectPeerFunc) (*tls.Config, error) {
	return &tls.Config{
		ClientAuth:            tls.RequireAnyClientCert,
		InsecureSkipVerify:    true,
		GetCertificate:        func(*tls.ClientHelloInfo) (*tls.Certificate, error) { return s.GetCertificate() },
		GetClientCertificate:  func(*tls.CertificateRequestInfo) (*tls.Certificate, error) { return s.GetCertificate() },
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error { return nil },
	}, nil
}

func (s *selfSignedTLSPeer) Close() error {
	s.renewTimer.Stop()
	return nil
}

func (s *selfSignedTLSPeer) WaitUntilReady(ctx context.Context) error {
	return nil
}
