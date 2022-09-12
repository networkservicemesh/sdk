// Copyright (c) 2020 Cisco and/or its affiliates.
//
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

// Package debug provides a very simple function that, if executed will replace the executable with
// dlv running the executable and listening on the port specified by an environment variable.
//
//	Example:
//	if err := debug.Self(); err != nil {
//	   fmt.Println(err)
//	}
package debug

import (
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// Dlv - name of Dlv executable, also used as default prefix to env variable, ie, DLV_
	Dlv = "dlv"
	// Listen - string used as default second part of env variable, ie, _PORT in DLV_PORT
	Listen       = "listen"
	envSeperator = "_"
)

// Self - replaces executable with dlv running the executable and listening on a port
//
//	envVariableParts - variadic list of 'pieces' to be transformed into the
//	                   env variable indicating where dlv should listen.
//	                   Default: If no envVariableParts are provided, env variable
//	                            defaults to []string{Dlv,Listen,path.Base(os.Args[0])}
//	                            which is DLV_PORT_<name of executable>
func Self(envVariableParts ...string) error {
	// Get the executable name.  Note: os.Args[0] may not be the absolute path of the executable, which is
	// why we get it this way
	executable, err := os.Executable()
	if err != nil {
		return errors.Errorf("unable to get excutable name: %+v", err)
	}
	// If you don't provide an env variable, we make a default choice
	if len(envVariableParts) == 0 {
		envVariableParts = []string{Dlv, Listen, path.Base(executable)}
	}
	dlvPortEnvVariable := envVariable(envVariableParts...)

	// Do we have that env variable?
	listen, exists := os.LookupEnv(dlvPortEnvVariable)
	if !exists {
		return errors.Errorf("Setting env variable %s to a valid dlv '--listen' value will cause the dlv debugger to execute this binary and listen as directed.", dlvPortEnvVariable)
	}

	// Is it a valid listen?
	split := strings.Split(listen, ":") // break on ':'
	if len(split) < 2 {
		return errors.Errorf("Unable to use value %q from env variable %s as a listen: missing ':' before port", listen, dlvPortEnvVariable)
	}
	if _, err = strconv.ParseUint(split[1], 10, 16); err != nil {
		return errors.Wrapf(err, "Unable to use value %q from env variable %s as a listen", listen, dlvPortEnvVariable)
	}

	// Do we have dlv?
	dlv, err := exec.LookPath("dlv")
	if err != nil {
		return errors.Wrap(err, "Unable to find dlv in your path")
	}

	// Marshal the new args
	args := []string{
		dlv,
		"--listen=" + listen,
		"--headless=true",
		"--api-version=2",
		"--accept-multiclient",
		"exec",
	}
	args = append(args, executable)
	if len(os.Args) > 1 {
		args = append(args, os.Args[1:]...)
	}

	// Remove the dlvPortEnvVariable from the environment variables
	origEnvv := os.Environ()
	var envv []string
	for _, env := range origEnvv {
		if !strings.HasPrefix(env, dlvPortEnvVariable) {
			envv = append(envv, env)
		}
	}
	// Make the syscall.Exec
	logrus.Infof("About to exec: \"%s %s\"", dlv, strings.Join(args[1:], " "))
	// About to debug this not working at host rather than container level
	return syscall.Exec(dlv, args, envv)
}

// envVariable - convert the envVariableParts to a proper env variable
func envVariable(envVariableParts ...string) string {
	env := strings.Join(envVariableParts, envSeperator)
	rgx := regexp.MustCompile("[[:^alnum:]]")
	env = rgx.ReplaceAllString(env, envSeperator)
	env = strings.ToUpper(env)
	return env
}
