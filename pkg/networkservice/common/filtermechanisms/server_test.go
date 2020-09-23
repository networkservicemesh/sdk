package filtermechanisms_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/srv6"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/filtermechanisms"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
)

func TestFilterMechanismsServer_Request(t *testing.T) {
	request := func() *networkservice.NetworkServiceRequest {
		return &networkservice.NetworkServiceRequest{
			MechanismPreferences: []*networkservice.Mechanism{
				{
					Cls:  cls.REMOTE,
					Type: srv6.MECHANISM,
				},
				{
					Cls:  cls.REMOTE,
					Type: vxlan.MECHANISM,
				},
				{
					Cls:  cls.LOCAL,
					Type: kernel.MECHANISM,
				},
				{
					Cls:  cls.LOCAL,
					Type: memif.MECHANISM,
				},
			},
		}
	}
	samples := []struct {
		Name         string
		ClientURL    *url.URL
		RegisterURLs []url.URL
		ClsResult    string
	}{
		{
			Name:      "Local mechanisms",
			ClientURL: &url.URL{Scheme: "tcp", Host: "test1"},
			RegisterURLs: []url.URL{
				{
					Scheme: "tcp",
					Host:   "test1",
				},
			},
			ClsResult: cls.LOCAL,
		},
		{
			Name:      "Remote mechanisms",
			ClientURL: &url.URL{Scheme: "tcp", Host: "test1"},
			ClsResult: cls.LOCAL,
		},
	}

	for _, sample := range samples {
		var registryServer registry.NetworkServiceEndpointRegistryServer
		s := filtermechanisms.NewServer(&registryServer)
		for _, u := range sample.RegisterURLs {
			registryServer.Register(context.Background(), &registry.NetworkServiceEndpoint{
				Url: u.String(),
			})
		}
		ctx := clienturlctx.WithClientURL(context.Background(), sample.ClientURL)
		req := request()
		_, err := s.Request(ctx, req)
		require.NoError(t, err)
		for _, m := range req.MechanismPreferences {
			require.Equal(t, sample.ClsResult, m.Cls, "filtermechanisms chain element should properly filter mechanisms")
		}
	}
}
