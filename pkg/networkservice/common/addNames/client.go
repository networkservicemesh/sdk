package addNames

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"os"
)

type addNamesClient struct{}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that adds pod, node and cluster names to request from environment variables
func NewClient() networkservice.NetworkServiceClient {
	return &addNamesClient{}
}

func (a *addNamesClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	varNames := map[string]string{"NODE_NAME": "NodeNameKey", "POD_NAME": "PodNameKey", "CLUSTER_NAME": "ClusterNameKey"}
	labels := request.GetRequestConnection().Labels
	for envName, labelName := range varNames {
		value := os.Getenv(envName) // TODO decide what to do about "" value here
		// TODO should i check map existence???
		// TODO warning when overwrite values
		labels[labelName] = value
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (a *addNamesClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
