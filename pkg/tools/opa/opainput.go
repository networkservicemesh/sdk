// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package opa

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"

	"github.com/pkg/errors"
	"google.golang.org/grpc/peer"

	"google.golang.org/grpc/credentials"
)

// PreparedOpaInput - converts model to map. It also puts auth_info in root of the map if it is presented in context.
func PreparedOpaInput(ctx context.Context, model interface{}) (map[string]interface{}, error) {
	result, err := convertToMap(model)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot convert %v to map", model)
	}
	p, ok := peer.FromContext(ctx)
	var cert *x509.Certificate
	if ok {
		cert = ParseX509Cert(p.AuthInfo)
	}
	var pemcert string
	if cert != nil {
		pemcert = pemEncodingX509Cert(cert)
	}
	result["auth_info"] = map[string]interface{}{
		"certificate": pemcert,
	}
	return result, nil
}

func pemEncodingX509Cert(cert *x509.Certificate) string {
	certpem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	return string(certpem)
}

// ParseX509Cert - parses x509 certificate from the passed credentials.AuthInfo
func ParseX509Cert(authInfo credentials.AuthInfo) *x509.Certificate {
	var peerCert *x509.Certificate

	switch v := authInfo.(type) {
	case *credentials.TLSInfo:
		if len(v.State.PeerCertificates) > 0 {
			peerCert = v.State.PeerCertificates[0]
		}
	case credentials.TLSInfo:
		if len(v.State.PeerCertificates) > 0 {
			peerCert = v.State.PeerCertificates[0]
		}
	}

	if tlsInfo, ok := authInfo.(*credentials.TLSInfo); ok {
		if len(tlsInfo.State.PeerCertificates) > 0 {
			peerCert = tlsInfo.State.PeerCertificates[0]
		}
	}
	return peerCert
}

func convertToMap(model interface{}) (map[string]interface{}, error) {
	jsonConn, err := json.Marshal(model)
	if err != nil {
		return nil, err
	}
	var rv map[string]interface{}
	if err := json.Unmarshal(jsonConn, &rv); err != nil {
		return nil, err
	}
	if rv == nil {
		rv = make(map[string]interface{})
	}
	return rv, nil
}
