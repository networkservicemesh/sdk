package swap

import (
	"context"
	"errors"
	"net"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type swapIPServer struct {
	externalIP string
}

func (i *swapIPServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	url := clienturl.ClientURL(ctx)
	if url == nil {
		return nil, errors.New("url is required for interdomain use-case")
	}
	dstIP, _, err := net.SplitHostPort(clienturl.ClientURL(ctx).Host)
	if err != nil {
		return nil, err
	}
	for _, m := range request.MechanismPreferences {
		if m.Cls == cls.REMOTE {
			request.Connection.Mechanism.Parameters[common.SrcIP] = i.externalIP
		}
	}
	if request.Connection.Mechanism != nil {
		request.Connection.Mechanism.Parameters[common.SrcIP] = i.externalIP
	}
	//	nsName, nseName := request.Connection.NetworkService, request.Connection.NetworkServiceEndpointName
	request.Connection.NetworkServiceEndpointName, request.Connection.NetworkService = interdomain.Target(request.Connection.NetworkServiceEndpointName), interdomain.Target(request.Connection.NetworkService)
	response, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	if response.Mechanism != nil {
		response.Mechanism.Parameters[common.DstIP] = dstIP
	}
	//	response.NetworkService = nsName
	//	response.NetworkServiceEndpointName = nseName
	return response, err
}

func (i *swapIPServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, connection)
}

func NewServer(externalIP net.IP) networkservice.NetworkServiceServer {
	if externalIP == nil {
		panic("externalIP should not be empty")
	}
	return &swapIPServer{
		externalIP: externalIP.String(),
	}
}
