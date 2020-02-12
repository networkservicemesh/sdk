package addNames

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"os"
)

type addNamesClient struct{}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that adds pod, node and cluster names to request from corresponding environment variables
func NewClient() networkservice.NetworkServiceClient {
	return &addNamesClient{}
}

func (a *addNamesClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	names := map[string]string{"NODE_NAME": "NodeNameKey", "POD_NAME": "PodNameKey", "CLUSTER_NAME": "ClusterNameKey"}
	conn := request.GetRequestConnection()
	if conn.Labels == nil {
		conn.Labels = make(map[string]string)
	}
	for envName, labelName := range names {
		value, exists := os.LookupEnv(envName)
		if exists {
			oldValue, isPresent := conn.Labels[labelName]
			if isPresent {
				logrus.Warningf("The label %s was already assigned to %s. Overwriting.", labelName, oldValue)
			}
			conn.Labels[labelName] = value
		} else {
			logrus.Warningf("Environment variable %s is not set. Skipping.", envName)
		}
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (a *addNamesClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
