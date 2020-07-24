// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
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

package prefixpool

import (
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

/* These variables set default path to the config file */
const (
	PrefixesFile            = "excluded_prefixes.yaml"
	NSMConfigDir            = "/var/lib/networkservicemesh/config"
	PrefixesFilePathDefault = NSMConfigDir + "/" + PrefixesFile
)

// Reader contains prefix pool structure and config file monitoring
type Reader struct {
	PrefixPool
	prefixesConfig *viper.Viper
	configPath     string
}

func (ph *Reader) init(prefixes []string) {
	ph.mutex.Lock()
	ph.prefixes = prefixes
	ph.mutex.Unlock()
	ph.connections = map[string]*connectionRecord{}
}

// NewPrefixPoolReader gets list of excluded prefixes from config file. Starts config file monitoring.
// Returns pointer to a struct that contains all information
func NewPrefixPoolReader(path string) *Reader {
	ph := &Reader{
		prefixesConfig: viper.New(),
		configPath:     path,
	}

	// setup watching the prefixes config file
	ph.prefixesConfig.SetDefault("prefixes", []string{})
	ph.prefixesConfig.SetConfigFile(ph.configPath)
	_ = ph.prefixesConfig.ReadInConfig()

	readPrefixes := func() {
		logrus.Infof("Reading excluded prefixes config file: %s", ph.configPath)
		prefixes := ph.prefixesConfig.GetStringSlice("prefixes")
		logrus.Infof("Excluded prefixes: %v", prefixes)
		ph.init(prefixes)
	}

	ph.prefixesConfig.OnConfigChange(func(fsnotify.Event) {
		logrus.Info("Excluded prefixes config file changed")
		readPrefixes()
	})
	ph.prefixesConfig.WatchConfig()
	readPrefixes()

	return ph
}
