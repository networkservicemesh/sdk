// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

// Package spire provides two simple functions:
//   - Start to start a SpireServer/SpireAgent for local testing
//   - AddEntry to add entries into the spire server
package spire

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/edwarnicke/exechelper"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

type contextKeyType string

const (
	logrusEntryKey contextKeyType = "LogrusEntry"

	healthCheckTimeout = 10 * time.Second
)

func withLog(parent context.Context) context.Context {
	entry := logrus.WithTime(time.Now())
	return context.WithValue(parent, logrusEntryKey, entry)
}

func logrusEntry(ctx context.Context) *logrus.Entry {
	if entryValue := ctx.Value(logrusEntryKey); entryValue != nil {
		if entry := entryValue.(*logrus.Entry); entry != nil {
			return entry
		}
	}
	return logrus.WithTime(time.Now())
}

// AddEntry - adds an entry to the spire server for parentID, spiffeID, and selector
//            parentID is usually the same as the agentID provided to Start()
func AddEntry(ctx context.Context, parentID, spiffeID, selector, federatesWith string) error {
	cmdStr := "spire-server entry create -parentID %s -spiffeID %s -selector %s"
	cmdStr = fmt.Sprintf(cmdStr, parentID, spiffeID, selector)
	if federatesWith != "" {
		cmdStr = fmt.Sprintf(cmdStr+" -federatesWith %s", federatesWith)
	}
	return exechelper.Run(cmdStr,
		exechelper.WithStdout(logrusEntry(ctx).WithField("cmd", cmdStr).WriterLevel(logrus.InfoLevel)),
		exechelper.WithStderr(logrusEntry(ctx).WithField("cmd", cmdStr).WriterLevel(logrus.WarnLevel)),
	)
}

// Start - start a spire-server and spire-agent with the given agentId
func Start(options ...Option) <-chan error {
	errCh := make(chan error, 4)
	defaultRoot, err := ioutil.TempDir("", "spire")
	if err != nil {
		errCh <- err
		close(errCh)
		return errCh
	}

	opt := &option{
		ctx:        withLog(context.Background()),
		agentID:    "spiffe://example.org/agent",
		agentConf:  fmt.Sprintf(spireAgentConfContents, defaultRoot),
		serverConf: fmt.Sprintf(spireServerConfContents, defaultRoot),
		spireRoot:  defaultRoot,
	}
	for _, o := range options {
		o(opt)
	}

	// Write the config files
	err = writeConfigFiles(opt.ctx, opt.agentConf, opt.serverConf, opt.spireRoot)
	if err != nil {
		errCh <- err
		close(errCh)
		return errCh
	}

	logrus.Infof("Env variable %s=%s are set", workloadapi.SocketEnv, "unix:"+spireDefaultEndpointSocket)
	if err = os.Setenv(workloadapi.SocketEnv, "unix:"+spireDefaultEndpointSocket); err != nil {
		errCh <- err
		close(errCh)
		return errCh
	}

	// Start the Spire Server
	spireCmd := fmt.Sprintf("spire-server run -config %s", path.Join(opt.spireRoot, spireServerConfFileName))
	spireServerErrCh := exechelper.Start(spireCmd,
		exechelper.WithDir(opt.spireRoot),
		exechelper.WithContext(opt.ctx),
		exechelper.WithStdout(logrusEntry(opt.ctx).WithField("cmd", "spire-server run").WriterLevel(logrus.InfoLevel)),
		exechelper.WithStderr(logrusEntry(opt.ctx).WithField("cmd", "spire-server run").WriterLevel(logrus.WarnLevel)),
	)
	select {
	case spireServerErr := <-spireServerErrCh:
		errCh <- spireServerErr
		close(errCh)
		return errCh
	default:
	}

	// Health check the Spire Server
	if err = execHealthCheck(opt.ctx,
		"spire-server healthcheck",
		exechelper.WithStdout(logrusEntry(opt.ctx).WithField("cmd", "spire-server healthcheck").WriterLevel(logrus.InfoLevel)),
		exechelper.WithStderr(logrusEntry(opt.ctx).WithField("cmd", "spire-server healthcheck").WriterLevel(logrus.WarnLevel)),
	); err != nil {
		errCh <- err
		close(errCh)
		return errCh
	}

	// Add Entries
	for _, entry := range opt.entries {
		if err = AddEntry(opt.ctx, opt.agentID, entry.spiffeID, entry.selector, ""); err != nil {
			errCh <- err
			close(errCh)
			return errCh
		}
	}

	// Add Federated Entries
	for _, entry := range opt.fEntries {
		for {
			if err = AddEntry(opt.ctx, opt.agentID, entry.spiffeID, entry.selector, entry.federatesWith); err != nil {
				logrus.Warn("error occurred while adding entry")

				// There will be an error, until we add a federated bundle. Retry.
				if entry.federatesWith != "" {
					time.Sleep(time.Second)
					continue
				}
				break
			}
			break
		}
	}

	// Get the SpireServers Token
	cmdStr := "spire-server token generate -spiffeID %s"
	cmdStr = fmt.Sprintf(cmdStr, opt.agentID)
	outputBytes, err := exechelper.Output(cmdStr,
		exechelper.WithStdout(logrusEntry(opt.ctx).WithField("cmd", cmdStr).WriterLevel(logrus.InfoLevel)),
		exechelper.WithStderr(logrusEntry(opt.ctx).WithField("cmd", cmdStr).WriterLevel(logrus.WarnLevel)),
	)
	if err != nil {
		errCh <- err
		close(errCh)
		return errCh
	}
	spireToken := strings.Replace(string(outputBytes), "Token:", "", 1)
	spireToken = strings.TrimSpace(spireToken)

	// Start the Spire Agent
	spireAgentErrCh := exechelper.Start("spire-agent run"+" -config "+spireAgentConfFilename+" -joinToken "+spireToken,
		exechelper.WithDir(opt.spireRoot),
		exechelper.WithContext(opt.ctx),
		exechelper.WithStdout(logrusEntry(opt.ctx).WithField("cmd", "spire-agent run").WriterLevel(logrus.InfoLevel)),
		exechelper.WithStderr(logrusEntry(opt.ctx).WithField("cmd", "spire-agent run").WriterLevel(logrus.WarnLevel)),
	)
	select {
	case spireAgentErr := <-spireAgentErrCh:
		errCh <- spireAgentErr
		close(errCh)
		return errCh
	default:
	}

	// Health check the Spire Agent
	if err = execHealthCheck(opt.ctx,
		"spire-agent healthcheck",
		exechelper.WithStdout(logrusEntry(opt.ctx).WithField("cmd", "spire-agent healthcheck").WriterLevel(logrus.InfoLevel)),
		exechelper.WithStderr(logrusEntry(opt.ctx).WithField("cmd", "spire-agent healthcheck").WriterLevel(logrus.WarnLevel)),
	); err != nil {
		errCh <- err
		close(errCh)
		return errCh
	}

	// Cleanup if either server we spawned dies
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func(errCh chan error, wg *sync.WaitGroup) {
		for {
			err, ok := <-spireServerErrCh
			if ok {
				errCh <- err
				continue
			}
			wg.Done()
			break
		}
	}(errCh, wg)
	go func(errCh chan error, wg *sync.WaitGroup) {
		for {
			err, ok := <-spireAgentErrCh
			if ok {
				errCh <- err
				continue
			}
			wg.Done()
			break
		}
	}(errCh, wg)
	go func(errCh chan error, wg *sync.WaitGroup) {
		wg.Wait()
		close(errCh)
	}(errCh, wg)
	return errCh
}

// writeConfigFiles - write config files into configRoot
func writeConfigFiles(ctx context.Context, agentConfig, serverConfig, spireRoot string) error {
	configFiles := map[string]string{
		spireServerConfFileName: serverConfig,
		spireAgentConfFilename:  agentConfig,
	}
	for configName, contents := range configFiles {
		filename := path.Join(spireRoot, configName)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			logrusEntry(ctx).Infof("Configuration file: %q not found, using defaults", filename)
			if err := os.MkdirAll(path.Dir(filename), 0o700); err != nil {
				return err
			}
			if err := ioutil.WriteFile(filename, []byte(contents), 0o600); err != nil {
				return err
			}
		}
	}
	return nil
}

func execHealthCheck(ctx context.Context, cmdStr string, options ...*exechelper.Option) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, healthCheckTimeout)
		defer cancel()
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := exechelper.Run(cmdStr, options...); err == nil {
				return nil
			}
		}
	}
}

// SpiffeIDFromContext - returns spiffe ID of the service from the peer context
func SpiffeIDFromContext(ctx context.Context) (spiffeid.ID, error) {
	p, ok := peer.FromContext(ctx)
	var cert *x509.Certificate
	if !ok {
		return spiffeid.ID{}, errors.New("fail to get peer from context")
	}
	cert = opa.ParseX509Cert(p.AuthInfo)
	if cert != nil {
		spiffeID, err := x509svid.IDFromCert(cert)
		if err == nil {
			return spiffeID, nil
		}
		return spiffeid.ID{}, errors.New("fail to get Spiffe ID from certificate")
	}
	return spiffeid.ID{}, errors.New("fail to get certificate from peer")
}
