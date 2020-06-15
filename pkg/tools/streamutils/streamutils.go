package streamutils

import "github.com/networkservicemesh/api/pkg/api/registry"

func ReadNetworkServiceList(stream registry.NetworkServiceRegistry_FindClient) []*registry.NetworkService {
	var result []*registry.NetworkService
	for msg, err := stream.Recv(); true; msg, err = stream.Recv() {
		if err != nil {
			break
		}
		result = append(result, msg)
	}
	return result
}

func ReadNetworkServiceEndpointList(stream registry.NetworkServiceEndpointRegistry_FindClient) []*registry.NetworkServiceEndpoint {
	var result []*registry.NetworkServiceEndpoint
	for msg, err := stream.Recv(); true; msg, err = stream.Recv() {
		if err != nil {
			break
		}
		result = append(result, msg)
	}
	return result
}
