package addNames

import (
	"context"
	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
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
	varNames := map[string]string{"NODE_NAME": "NodeNameKey", "POD_NAME": "PodNameKey", "CLUSTER_NAME": "ClusterNameKey"}
	if request.GetRequestConnection().Labels == nil {
		request.GetRequestConnection().Labels = make(map[string]string)
	}
	for envName, labelName := range varNames {
		value, isPresent := os.LookupEnv(envName)
		if !isPresent {
			return nil, errors.Errorf("Environment variable %s is not set", envName)
		}
		// TODO do i need warning when overwrite values???
		request.GetRequestConnection().Labels[labelName] = value
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (a *addNamesClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
