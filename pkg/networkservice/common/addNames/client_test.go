package addNames

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"os"
	"reflect"
	"testing"
)

type addNamesTestData struct {
	name    string
	envs    map[string]string
	request *networkservice.NetworkServiceRequest
	want    *networkservice.Connection
}

var tests = []addNamesTestData{
	{
		"labels not present",
		map[string]string{
			"NODE_NAME":    "AAA",
			"POD_NAME":     "BBB",
			"CLUSTER_NAME": "CCC",
		},
		&networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{},
		},
		&networkservice.Connection{
			Labels: map[string]string{
				"NodeNameKey":    "AAA",
				"PodNameKey":     "BBB",
				"ClusterNameKey": "CCC",
			},
		},
	},
}

func Test_addNamesClient_Request(t *testing.T) {
	server := next.NewNetworkServiceClient(NewClient())
	for _, testData := range tests {
		for name, value := range testData.envs {
			err := os.Setenv(name, value)
			if err != nil {
				t.Errorf("%s: addNamesClient.Request() unable to set up environment variable: %v", testData.name, err)
			}
		}

		got, _ := server.Request(context.Background(), testData.request)
		if !reflect.DeepEqual(got, testData.want) {
			t.Errorf("%s: addNamesClient.Request() = %v, want %v", testData.name, got, testData.want)
		}

		for name := range testData.envs {
			err := os.Unsetenv(name)
			if err != nil {
				t.Errorf("%s: addNamesClient.Request() unable to unset environment variable: %v", testData.name, err)
			}
		}
	}
}
