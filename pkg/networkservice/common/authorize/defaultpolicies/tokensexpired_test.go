package defaultpolicies

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/opautils"

	"github.com/open-policy-agent/opa/rego"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getConnectionWithTokens(tokensExpireTime []time.Time) (*networkservice.Connection, error) {
	rv := &networkservice.Connection{
		Path: &networkservice.Path{
			Index: 0,
			PathSegments: []*networkservice.PathSegment{},
		},
	}

	for _, expireTimes := range tokensExpireTime {
		token, err := generateTokenWithExpireTime(expireTimes)
		if err != nil {
			return nil, err
		}
		rv.Path.PathSegments = append(rv.Path.PathSegments, &networkservice.PathSegment{
			Token: token,
		})
	}

	return rv, nil
}

func TestNoTokensExpiredPolicy(t *testing.T){
	suits := []struct {
		name     string
		tokensExpireTime   []time.Time
		isNotExpired bool
	}{
		{
			name:  "simple positive test with one token",
			tokensExpireTime: []time.Time{
				time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC),
			},
			isNotExpired: true,
		},
		{
			name:    "negative test with expired/not expired tokens",
			tokensExpireTime: []time.Time{
				time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC),
				time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC),
			},
			isNotExpired: false,
		},
	}

	policyBytes, err := ioutil.ReadFile("tokensexpired.rego")
	require.Nil(t, err)

	p, err := rego.New(
		rego.Query("data.defaultpolicies.no_any_expired_tokens"),
		rego.Module("tokensexpired.rego", string(policyBytes))).PrepareForEval(context.Background())
	require.Nilf(t, err, "failed to create new rego policy: %v", err)

	for i := range suits {
		s := suits[i]

		conn, err := getConnectionWithTokens(s.tokensExpireTime)
		require.Nil(t, err)

		input, err := opautils.PreparedOpaInput(conn, nil, opautils.Request, opautils.Client)
		require.Nil(t, err)

		t.Run(s.name, func(t *testing.T) {
			checkResult := func(err error) {
				if s.isNotExpired {
					require.Nil(t, err)
					return
				}

				require.NotNil(t, err)
				s, ok := status.FromError(err)
				require.True(t, ok, "error without error status code")
				require.Equal(t, s.Code(), codes.PermissionDenied, "wrong error status code")
			}

			err = opautils.CheckPolicy(context.Background(), &p, input)
			checkResult(err)
		})
	}
}