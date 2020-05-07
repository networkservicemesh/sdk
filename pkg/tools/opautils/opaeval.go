package opautils

import (
	"context"
	"errors"
	"github.com/open-policy-agent/opa/rego"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CheckPolicy(ctx context.Context, p *rego.PreparedEvalQuery, input interface{}) error {
	rs, err := p.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	hasAccess, err := hasAccess(rs)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if !hasAccess {
		return status.Error(codes.PermissionDenied, "no sufficient privileges to call Request")
	}

	return nil
}

func hasAccess(rs rego.ResultSet) (bool, error) {
	for _, r := range rs {
		for _, e := range r.Expressions {
			t, ok := e.Value.(bool)
			if !ok {
				return false, errors.New("policy contains non boolean expression")
			}

			if !t {
				return false, nil
			}
		}
	}

	return true, nil
}
