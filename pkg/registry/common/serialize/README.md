# Functional requirements

`Register`, `Unregister` events for the same `NetworkService.Name`, `NetworkServiceEndpoint.Name` should be executed in
registry chain serially.

# Implementation

## serializeNSServer, serializeNSClient, serializeNSEServer, serializeNSEclient

It is just the same as serialize chain elements for the network service chain. Please see [serialize](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/serialize)
for more details.
