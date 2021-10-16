// Copyright (c) 2021 Cisco and/or its affiliates.
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

// Package trimpath provides a simple pair of chain elements, one for the server side, one for the client side.
// If trimpath.NewServer() is present in a chain, it will 'trim' the path down to ending with that server *unless*
// it is a 'passthrough' server and the client it encapsulates contains trimpath.NewClient()
package trimpath
