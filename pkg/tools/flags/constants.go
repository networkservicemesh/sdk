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

// Package flags provides utility functions for flags for NSM cmds
package flags

const (
	// EnvPrefix - prefix for NSM environment variables
	EnvPrefix = "NSM"

	// NameKey - key for flag for name of NSM cmd
	NameKey = "name"
	// NameShortHand - shorthand for flag for name of NSM cmd
	NameShortHand = "n"
	// NameUsageDefault - default usage for flag for name of NSM cmd
	NameUsageDefault = "Name for this endpoint"

	// SpiffeAgentURLKey - key for flag for SpiffeAgentURL
	SpiffeAgentURLKey = "spiffe-agent-url"
	// SpiffeAgentURLShortHand - shorthand for flag for SpiffeAgentURL
	SpiffeAgentURLShortHand = "s"
	// SpiffeAgentURLSchemeDefault - default scheme for SpiffeAgentURL
	SpiffeAgentURLSchemeDefault = "unix"
	// SpiffeAgentURLPathDefault - default path for SpiffeAgentURL
	SpiffeAgentURLPathDefault = "/run/spire/sockets/agent.sock"
	// SpiffeAgentURLUsageDefault - default usage for flag for SpiffeAgentURL
	SpiffeAgentURLUsageDefault = "URL to Spiffe Agent"

	// ListenOnURLKey - key for flag for ListenOnURL
	ListenOnURLKey = "listen-on-url"
	// ListenOnURLShortHand - shorthand for flag for ListenOnURL
	ListenOnURLShortHand = "l"
	// ListenOnURLSchemeDefault - default scheme for ListenOnURL
	ListenOnURLSchemeDefault = "unix"
	// ListenOnURLPathDefault - default path for ListenOnURL
	ListenOnURLPathDefault = "/listen.on.socket"
	// ListenOnURLUsageDefault - default usage for ListenOnURL
	ListenOnURLUsageDefault = "URL to listen for incoming networkservicemesh RPC calls"

	// ConnectToURLKey - key for flag for ConnectToURL
	ConnectToURLKey = "connect-to-url"
	// ConnectToURLShortHand - shorthand for flag for ConnectToURL
	ConnectToURLShortHand = "c"
	// ConnectToURLSchemeDefault - default scheme for ConnectToURL
	ConnectToURLSchemeDefault = "unix"
	// ConnectToURLPathDefault - default path for ConnectToURL
	ConnectToURLPathDefault = "/connect.to.socket"
	// ConnectToURLUsageDefault - default usage for ConnectToURL
	ConnectToURLUsageDefault = "URL to make an outgoing networkservicemesh RPC calls"

	// BaseDirKey - key for basedir flag
	BaseDirKey = "base-dir"
	// BaseDirShortHand - shorthand for basedir flag
	BaseDirShortHand = "b"
	// BaseDirDefault - default basedir
	BaseDirDefault = "./"
	// BaseDirUsageDefault - default basedir usage
	BaseDirUsageDefault = "BaseDir for file sockets"

	// AuthzPolicyFileKey - key for flag for authzPolicyFile
	AuthzPolicyFileKey = "authz-policy-file"
	// AuthzPolicyFileShortHand - shorthand for flag for authzPolicyFile
	AuthzPolicyFileShortHand = "a"
	// AuthzPolicyFileDefault - default authzPolicyFile
	AuthzPolicyFileDefault = "/etc/nsm/authz.rego"
	// AuthzPolicyFileUsageDefault - default authzPolicyFile usage
	AuthzPolicyFileUsageDefault = "File containing OPA Policy"
)
