// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package selector provides a selection by any algorithm any item
package selector

// Decider interface provides selection from options
type Decider interface {
	Decide(options ...interface{}) (interface{}, error)
}

// OptionsProvider provides options for Decider
type OptionsProvider interface {
	GetOptions() ([]interface{}, error)
}

// Selector provides item selection via some Decider and OptionsProvider
type Selector struct {
	decider         Decider
	optionsProvider OptionsProvider
}

// Select return selected item
func (s *Selector) Select() (interface{}, error) {
	options, err := s.optionsProvider.GetOptions()
	if err != nil {
		return "", err
	}
	return s.decider.Decide(options...)
}

// New construct new Selector
func New(decider Decider, optionsProvider OptionsProvider) *Selector {
	return &Selector{
		decider:         decider,
		optionsProvider: optionsProvider,
	}
}
