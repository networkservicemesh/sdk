package addNames

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"os"
	"reflect"
	"testing"
)

func Test_addNamesClient_Request(t *testing.T) {

	envs := map[string]string{
		"NODE_NAME":    "AAA",
		"POD_NAME":     "BBB",
		"CLUSTER_NAME": "CCC",
	}

	for name, value := range envs {
		err := os.Setenv(name, value)
		if err != nil {
			t.Errorf("addNamesClient.Request() unable to set up environment variable: %v", err)
		}
	}

	want := &networkservice.Connection{
		Id: "1",
		Mechanism: &networkservice.Mechanism{
			Type: kernel.MECHANISM,
		},
		Context: &networkservice.ConnectionContext{
			IpContext: &networkservice.IPContext{
				DstIpAddr: "172.16.1.2",
			},
		},
		Labels: map[string]string{
			"NodeNameKey":    "AAA",
			"PodNameKey":     "BBB",
			"ClusterNameKey": "CCC",
		},
	}

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

	server := next.NewNetworkServiceClient(NewClient())

	got, _ := server.Request(context.Background(), request)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("addNamesClient.Request() = %v, want %v", got, want)
	}

}
