package addNames

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"reflect"
	"testing"
)

func Test_roundRobinSelector_SelectEndpoint(t *testing.T) {
	addNamesClient := NewClient()

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "1",
			Mechanism: &networkservice.Mechanism{
				Type: kernel.MECHANISM,
			},
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					DstIpAddr: "172.16.1.2",
				},
			},
		},
	}

}
