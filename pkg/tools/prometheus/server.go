// Copyright (c) 2025 Nordix Foundation.
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
	"crypto/x509"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// Server is a server type for exposing Prometheus metrics
type Server struct {
	tlsConfig        *tls.Config
	certHandler      *certHandler
	listenOn         string
	certFile         string
	keyFile          string
	caFile           string
	monitorCert      bool
	headerTimeout    time.Duration
	maxBindThreshold time.Duration
}

// Option is an option pattern for prometheus server
type Option func(s *Server)

// WithCustomCert sets the certificate and key to use for TLS
func WithCustomCert(certFile, keyFile string) Option {
	return func(s *Server) {
		s.certFile = certFile
		s.keyFile = keyFile
	}
}

// WithCustomCA sets the CA file to use for mTLS
func WithCustomCA(caFile string) Option {
	return func(s *Server) {
		s.caFile = caFile
	}
}

// WithHeaderTimeout sets the header timeout for the prometheus server
func WithHeaderTimeout(headerTimeout time.Duration) Option {
	return func(s *Server) {
		s.headerTimeout = headerTimeout
	}
}

// WithCertificateMonitoring enables monitoring for certificate renewals
func WithCertificateMonitoring(monitorCert bool) Option {
	return func(s *Server) {
		s.monitorCert = monitorCert
	}
}

// WithMaxBindThreshold sets the maximum threshold for binding retries
func WithMaxBindThreshold(maxBindThreshold time.Duration) Option {
	return func(s *Server) {
		s.maxBindThreshold = maxBindThreshold
		if maxBindThreshold <= 0 {
			log.FromContext(context.Background()).Warn("maxBindThreshold should be greater than 0, using default value of 120 seconds")
			s.maxBindThreshold = 120 // default to 120 seconds
		}
	}
}

// NewServer creates a new prometheus server instance
func NewServer(listenOn string, options ...Option) *Server {
	server := &Server{
		listenOn:         listenOn,
		certFile:         "",
		keyFile:          "",
		caFile:           "",
		monitorCert:      false,
		headerTimeout:    5 * time.Second,
		maxBindThreshold: 120 * time.Second, // default to 120 seconds
	}
	for _, opt := range options {
		opt(server)
	}

	return server
}

// ListenAndServe gathers the certificate and initiates the Server to begin handling incoming requests
func (s *Server) ListenAndServe(ctx context.Context, cancel context.CancelFunc) {
	log.FromContext(ctx).Debugf("new metrics server created with parameters listenOn: '%v', certFile: '%v', keyFile: '%v', caFile: '%v', headerTimeout: '%v', monitorCert: '%v'",
		s.listenOn, s.certFile, s.keyFile, s.caFile, s.headerTimeout, s.monitorCert)
	isTLSEnabled := s.certFile != "" && s.keyFile != ""
	if isTLSEnabled {
		s.createTLSConfig(ctx)
	} else {
		log.FromContext(ctx).Warn("using insecure metrics server")
		s.tlsConfig = nil
	}

	if isTLSEnabled && s.monitorCert {
		err := s.monitorCertificate(ctx)
		if err != nil {
			log.FromContext(ctx).Error(err.Error())
		}
	}

	go func() {
		err := s.start(ctx, isTLSEnabled)
		if err != nil {
			log.FromContext(ctx).Error(err.Error())
		}
	}()
}

func (s *Server) start(ctx context.Context, isTLSEnabled bool) error {
	log.FromContext(ctx).Info("start metrics server on ", s.listenOn)

	server := &http.Server{
		Addr:              s.listenOn,
		TLSConfig:         s.tlsConfig,
		ReadHeaderTimeout: s.headerTimeout,
	}

	http.Handle("/metrics", promhttp.Handler())

	serverCtx, cancel := context.WithCancel(ctx)
	var ListenAndServeErr error

	go func() {
		i := 1 // retry interval in seconds
		totalRetentionTime := 0
		for {
			if isTLSEnabled {
				ListenAndServeErr = server.ListenAndServeTLS("", "")
			} else {
				ListenAndServeErr = server.ListenAndServe()
			}
			if ListenAndServeErr != nil {
				if strings.Contains(ListenAndServeErr.Error(), syscall.EADDRINUSE.Error()) {
					// address is already in use, wait for a while and then retry
					time.Sleep(time.Duration(i) * time.Second)
					totalRetentionTime += i
					i *= 2 // Exponential backoff
					if time.Duration(totalRetentionTime)*time.Second > s.maxBindThreshold {
						log.FromContext(ctx).Warnf("metrics server failed to start on %s after multiple attempts: %s", s.listenOn, ListenAndServeErr)
						cancel()
					}
				} else {
					cancel()
				}
			} else {
				break
			}
		}
	}()

	if ListenAndServeErr == nil {
		log.FromContext(ctx).Debugf("metrics server started successfully on %s", s.listenOn)
	}

	<-serverCtx.Done()

	if ListenAndServeErr != nil {
		return errors.Errorf("failed to ListenAndServe on metrics server: %s", ListenAndServeErr)
	}

	log.FromContext(ctx).Debugf("metrics server shuting down on %s", s.listenOn)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()
	defer func() { _ = server.Close() }() // just to be sure the server is closed

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		return errors.Errorf("failed to shutdown metrics server: %s", err)
	}

	return nil
}

func (s *Server) createTLSConfig(ctx context.Context) {
	log.FromContext(ctx).Debug("create TLS config for metrics server")

	s.tlsConfig = &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	s.certHandler = &certHandler{}
	err := s.certHandler.LoadCertificate(s.certFile, s.keyFile, s.caFile)
	if err != nil {
		log.FromContext(ctx).Errorf("error creating tls config: %v", err)
	}
	s.tlsConfig.GetCertificate = s.certHandler.GetCertificate
	if s.caFile != "" {
		log.FromContext(ctx).Debug("enable client authentication for metrics server")

		s.tlsConfig.ClientCAs = s.certHandler.caCertPool
		s.tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}
}

type certHandler struct {
	cert       *tls.Certificate
	caCertPool *x509.CertPool
	mu         sync.RWMutex
}

func (certHandler *certHandler) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	certHandler.mu.RLock()
	defer certHandler.mu.RUnlock()
	return certHandler.cert, nil
}

func (certHandler *certHandler) LoadCertificate(certFile, keyFile, caFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return errors.Errorf("error loading custom certificate and key: %v", err)
	}
	certHandler.mu.Lock()
	defer certHandler.mu.Unlock()
	certHandler.cert = &cert

	if caFile != "" {
		err = certHandler.LoadCertificateAuthority(caFile)
		if err != nil {
			return errors.Errorf("error loading custom certificate authority: %v", err)
		}
	}
	return nil
}

func (certHandler *certHandler) LoadCertificateAuthority(caFile string) error {
	cleanCaFile := filepath.Clean(caFile)
	if !strings.HasPrefix(cleanCaFile, "/run/secrets/") {
		return errors.Errorf("invalid CA file path")
	}

	caCert, err := os.ReadFile(cleanCaFile)
	if err != nil {
		return errors.Errorf("failed to read CA certificate: %s", err)
	}

	certHandler.caCertPool = x509.NewCertPool()
	ok := certHandler.caCertPool.AppendCertsFromPEM(caCert)
	if !ok {
		return errors.Errorf("failed to add CA certificate to the pool")
	}

	return nil
}

func (s *Server) monitorCertificate(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Errorf("failed to create new watcher: %s", err)
	}

	certFolder := filepath.Dir(s.certFile)
	certFileName := filepath.Join(certFolder, "..data")

	go func() {
		defer func() {
			if e := watcher.Close(); e != nil {
				log.FromContext(ctx).Errorf("error closing watcher: %v", e)
			}
		}()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.FromContext(ctx).Error("certificate watcher event channel closed")
					return
				}
				log.FromContext(ctx).Debugf("certificate watcher event: %v", event)
				if event.Name == certFileName && event.Op&fsnotify.Create == fsnotify.Create {
					log.FromContext(ctx).Infof("certificate file '%s' was modified, reloading certificate", event.Name)
					e := s.certHandler.LoadCertificate(s.certFile, s.keyFile, s.caFile)
					if e != nil {
						log.FromContext(ctx).Errorf("failed to reload metrics server certificate: %v", e)
					}
				}

			case e, ok := <-watcher.Errors:
				if !ok {
					log.FromContext(ctx).Errorf("certificate watcher event channel closed: %v", e)
					return
				}
				log.FromContext(ctx).Errorf("certificate watcher error: %v", e)

			case <-ctx.Done():
				log.FromContext(ctx).Info("stopping certificate watcher due to context cancellation")
				return
			}
		}
	}()

	err = watcher.Add(certFolder)
	if err != nil {
		return errors.Errorf("failed to add certificate folder to file watcher: %v", err)
	}

	return nil
}
