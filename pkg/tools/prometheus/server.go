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

// Package prometheus provides a set of utilities for assisting with Prometheus data
package prometheus

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// ListenAndServe gathers the certificate and initiates the server to begin handling incoming requests.
func ListenAndServe(ctx context.Context, listenOn string, headerTimeout time.Duration, cancel context.CancelFunc) {
	metricsServer := server{
		ListenOn:      listenOn,
		HeaderTimeout: headerTimeout,
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		log.FromContext(ctx).Fatalf("error getting x509 source: %v", err.Error())
	}
	tlsConfig.GetCertificate = tlsconfig.GetCertificate(source)
	metricsServer.TLSConfig = tlsConfig

	select {
	case <-ctx.Done():
		err = source.Close()
		log.FromContext(ctx).Errorf("unable to close x509 source: %v", err.Error())
	default:
	}

	go func() {
		err := metricsServer.start(ctx)
		if err != nil {
			log.FromContext(ctx).Error(err.Error())
			cancel()
		}
	}()
}

type server struct {
	TLSConfig     *tls.Config
	ListenOn      string
	HeaderTimeout time.Duration
}

func (s *server) start(ctx context.Context) error {
	log.FromContext(ctx).Info("Start metrics server on ", s.ListenOn)

	server := &http.Server{
		Addr:              s.ListenOn,
		TLSConfig:         s.TLSConfig,
		ReadHeaderTimeout: s.HeaderTimeout,
	}

	http.Handle("/metrics", promhttp.Handler())

	serverCtx, cancel := context.WithCancel(ctx)
	var ListenAndServeErr error

	go func() {
		ListenAndServeErr = server.ListenAndServeTLS("", "")
		if ListenAndServeErr != nil {
			cancel()
		}
	}()

	<-serverCtx.Done()

	if ListenAndServeErr != nil {
		return errors.Errorf("failed to ListenAndServe on metrics server: %s", ListenAndServeErr)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		return errors.Errorf("failed to shutdown metrics server: %s", err)
	}

	return nil
}
