package interdomainurl

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type interdomainURLServer struct {
}

func (i *interdomainURLServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	nseName := request.GetConnection().GetNetworkServiceEndpointName()
	if nseName == "" {
		return nil, errors.New("NSE is not selected")
	}
	remoteURL := interdomain.Domain(nseName)
	u, err := url.Parse(remoteURL)
	if err != nil {
		return nil, errors.Wrap(err, "selected NSE has wrong name. Make sure that proxy-registry has handled NSE")
	}
	request.GetConnection().NetworkServiceEndpointName = interdomain.Target(nseName)
	return next.Server(ctx).Request(clienturl.WithClientURL(ctx, u), request)
}

func (i *interdomainURLServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	nseName := connection.GetNetworkServiceEndpointName()
	if nseName == "" {
		return nil, errors.New("NSE is not selected")
	}
	remoteURL := interdomain.Domain(nseName)
	u, err := url.Parse(remoteURL)
	if err != nil {
		return nil, errors.Wrap(err, "selected NSE has wrong name. Make sure that proxy-registry has handled NSE")
	}
	connection.NetworkServiceEndpointName = interdomain.Target(nseName)
	return next.Server(ctx).Close(clienturl.WithClientURL(ctx, u), connection)
}

func NewServer() networkservice.NetworkServiceServer {
	return &interdomainURLServer{}
}
