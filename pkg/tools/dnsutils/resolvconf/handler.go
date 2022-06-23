// Copyright (c) 2022 Cisco and/or its affiliates.
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

// Package resolvconf configures resolv.conf file
package resolvconf

import (
	"context"
	"io/ioutil"
	"os"
	"sync"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

var once sync.Once

type resolvConfigHandler struct {
	chainContext           context.Context
	resolveConfigPath      string
	storedResolvConfigPath string
	defaultNameServerIP    string
}

func (h *resolvConfigHandler) ServeDNS(ctx context.Context, rp dns.ResponseWriter, m *dns.Msg) {
	if m == nil {
		dns.HandleFailed(rp, m)
		return
	}

	once.Do(h.initialize)
	next.Handler(ctx).ServeDNS(ctx, rp, m)
}

// NewDNSHandler creates a new dns handler that configures resolv.conf file
func NewDNSHandler(opts ...Option) dnsutils.Handler {
	handler := &resolvConfigHandler{
		chainContext:        context.Background(),
		defaultNameServerIP: "127.0.0.1",
		resolveConfigPath:   "/etc/resolv.conf",
	}

	for _, o := range opts {
		o(handler)
	}

	return handler
}

func (h *resolvConfigHandler) restoreResolvConf() {
	originalResolvConf, err := ioutil.ReadFile(h.storedResolvConfigPath)
	if err != nil || len(originalResolvConf) == 0 {
		return
	}
	_ = os.WriteFile(h.resolveConfigPath, originalResolvConf, os.ModePerm)
}

func (h *resolvConfigHandler) storeOriginalResolvConf() {
	if _, err := os.Stat(h.storedResolvConfigPath); err == nil {
		return
	}
	originalResolvConf, err := ioutil.ReadFile(h.resolveConfigPath)
	if err != nil {
		return
	}
	_ = ioutil.WriteFile(h.storedResolvConfigPath, originalResolvConf, os.ModePerm)
}

func (h *resolvConfigHandler) initialize() {
	h.restoreResolvConf()

	r, err := openResolveConfig(h.resolveConfigPath)
	if err != nil {
		log.FromContext(h.chainContext).Errorf("An error during open resolve config: %v", err.Error())
		return
	}

	h.storeOriginalResolvConf()

	r.SetValue(NameserverProperty, h.defaultNameServerIP)

	if err = r.Save(); err != nil {
		log.FromContext(h.chainContext).Errorf("An error during save resolve config: %v", err.Error())
		return
	}
}
