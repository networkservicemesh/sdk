package matchutils

import (
	"reflect"
	"strings"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

func MatchNetworkServices(left, right *registry.NetworkService) bool {
	return (left.Name == "" || left.Name == right.Name) &&
		(left.Payload == "" || left.Payload == right.Payload) &&
		(left.Matches == nil || reflect.DeepEqual(left.Matches, right.Matches))
}

func MatchNetworkServiceEndpoints(left, right *registry.NetworkServiceEndpoint) bool {
	return (left.Name == "" || matchString(right.Name, left.Name)) &&
		(left.NetworkServiceLabels == nil || reflect.DeepEqual(left.NetworkServiceLabels, right.NetworkServiceLabels)) &&
		(left.ExpirationTime == nil || left.ExpirationTime.Seconds == right.ExpirationTime.Seconds) &&
		(left.NetworkServiceName == nil || reflect.DeepEqual(left.NetworkServiceName, right.NetworkServiceName)) &&
		(left.Url == "" || left.Url == right.Url)
}

func matchString(left, right string) bool {
	return strings.Contains(left, right)
}
