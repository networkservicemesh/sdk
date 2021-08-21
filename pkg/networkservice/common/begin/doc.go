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

/*
Package begin provides a chain element that can be put at the beginning of the chain, after Connection.Id has been set
but before any chain elements that would mutate the Connection on the return path.
the begin.New{Client,Server}() guarantee:

Scope

All Request() or Close() events are scoped to a particular Connection, uniquely identified by its Connection.Id

Exclusivity

Only one event is processed for a Connection.Id at a time

Order

Events for a given Connection.Id are processed in the order in which they are received

Close Correctness

When a Close(Connection) event is received, begin will replace the Connection provided with the last Connection
successfully returned from the chain for Connection.Id

Midchain Originated Events

A midchain element may originate a Request() or Close() event to be processed
from the beginning of the chain (Timeout, Refresh,Heal):

	errCh := begin.FromContext(ctx).Request()
	errCh := begin.FromContext(ctx).Close()

errCh will receive any error from the firing of the event, and will be closed after the event has fully
processed.

Note: if a chain is a server chain continued by a client chain, the beginning of the chain is at the beginning of
the server chain, even if there is a subsequent begin.NewClient() in the client chain.

Optionally you may use the CancelContext(context.Context) option:

	begin.FromContext(ctx).Request(CancelContext(cancelContext))
	begin.FromContext(ctx).Close(CancelContext(cancelContext))

If cancelContext is canceled prior to the processing of the event, the event processing will be skipped,
and the errCh returned simply closed.

Midchain Originated Request Event

Example:

	begin.FromContext(ctx).Request()

will use the networkservice.NetworkServiceRequest from the chain's last successfully completed Request() event
with networkservice.NetworkServiceRequest.Connection replaced with the Connection returned by the chain's last
successfully completed Request() event

Chain Placement

begin.New{Server/Client} should always proceed any chain element which:
- Maintains state
- Mutates the Connection object along the return path of processing a Request() event.

Reasoning

networkservice.NetworkService{Client,Server} processes two kinds of events:
  - Request()
  - Close()
Each Request() or Close() event is scoped to a networkservice.Connection, which can be uniquely identified by its Connection.Id

For a given Connection.Id, at most one event can be processed at a time (exclusivity).
For a given Connection.Id, events must be processed in the order they were received (order).
For Close(), the Connection passed to it must be identical to the last one returned by the chain to insure all state
is correctly cleared (close correctness).

Typically, a chain element receives a Request() or Close() event from the element before it in the chain
and sends a Request() or Close() and either terminates processing returning an error, or sends a Request() or Close()
event to the next element in the chain.

There are some circumstances in which a Request() or Close() event needs to be originated by a chain element
in the middle of the chain, but processed from the beginning of the chain.  Examples include (but are not limited to):
  - A server timing out an expired Connection
  - A client refreshing a Connection so that it does not expire
  - A client healing from a lost Connection
In all of these cases, the Request() or Close() event should be processed starting at the beginning of the chain, to ensure
that all of the proper side effects occur within the chain.
*/
package begin
