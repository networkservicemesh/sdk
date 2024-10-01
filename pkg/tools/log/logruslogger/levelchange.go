// Copyright (c) 2024 OpenInfra Foundation Europe. All rights reserved.
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

package logruslogger

import (
	"context"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// SetupLevelChangeOnSignal sets the loglevel to the one specified in the map when a signal assotiated to it arrives
func SetupLevelChangeOnSignal(ctx context.Context, signals map[os.Signal]logrus.Level) {
	sigChannel := make(chan os.Signal, len(signals))
	for sig := range signals {
		signal.Notify(sigChannel, sig)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				signal.Stop(sigChannel)
				close(sigChannel)
				return
			case sig := <-sigChannel:
				lvl := signals[sig]
				log.FromContext(ctx).Infof("Setting log level to '%s'", lvl.String())
				logrus.SetLevel(lvl)
			}
		}
	}()
}
