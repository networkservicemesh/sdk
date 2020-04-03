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

package flags

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// EnvReplacer - replacer that replaces '-' with '_'
var EnvReplacer *strings.Replacer = strings.NewReplacer("-", "_")

// KeyToEnvVariable - translate a flag key to a viper ENV variable
func KeyToEnvVariable(key string) string {
	key = strings.ToUpper(key)
	key = EnvReplacer.Replace(key)
	return fmt.Sprintf("%s_%s", EnvPrefix, key)
}

// FromEnv - wires up the provided flags with viper and returns a function that when invoked
//           will populate the flags from Env variables
//           envPrefix - prefix for env variables
//           envReplacer - replacer for env variables, for example replacing '-' with '_'
func FromEnv(envPrefix string, envReplacer *strings.Replacer, flagSet *pflag.FlagSet) func() {
	_ = viper.BindPFlags(flagSet)
	viper.AutomaticEnv()
	viper.SetEnvPrefix(envPrefix)
	if envReplacer != nil {
		viper.SetEnvKeyReplacer(envReplacer)
	}
	return func() {
		presetRequiredFlags(flagSet)
	}
}

func presetRequiredFlags(flagSet *pflag.FlagSet) {
	flagSet.VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			err := flagSet.Set(f.Name, viper.GetString(f.Name))
			if err != nil {
				fmt.Println("Error:", err.Error())
			}
		}
	})
}
