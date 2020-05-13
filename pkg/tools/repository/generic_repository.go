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

package repository

import (
	"github.com/cheekybits/genny/generic"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

type Generic generic.Type

type memoryGenericRepository struct {
	items map[string]*Generic
	serialize.Executor
	getId func(*Generic) string
}

func (g *memoryGenericRepository) Get(id string) *Generic {
	var result *Generic
	<-g.AsyncExec(func() {
		result = g.items[id]
	})
	return result
}

func (g *memoryGenericRepository) Put(item *Generic) {
	<-g.AsyncExec(func() {
		g.items[g.getId(item)] = item
	})
}

func (g *memoryGenericRepository) Delete(id string) {
	<-g.AsyncExec(func() {
		delete(g.items, id)
	})
}

func (g *memoryGenericRepository) GetAll() []*Generic {
	return g.GetAllByFilter(func(service *Generic) bool {
		return true
	})
}

func (g *memoryGenericRepository) GetAllByFilter(filter func(service *Generic) bool) []*Generic {
	var items []*Generic
	<-g.AsyncExec(func() {
		for _, v := range g.items {
			if filter(v) {
				items = append(items, v)
			}
		}
	})
	return items
}

type GenericRepository interface {
	AsyncExec(f func()) <-chan struct{}
	Get(string) *Generic
	Put(*Generic)
	Delete(string)
	GetAll() []*Generic
	GetAllByFilter(func(service *Generic) bool) []*Generic
}

func NewGenericMemoryRepository(getId func(*Generic) string) GenericRepository {
	return &memoryGenericRepository{
		getId: getId,
		items: map[string]*Generic{},
	}
}
