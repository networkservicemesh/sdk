package main

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	api_kernel "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/open-policy-agent/opa/rego"
	"log"
)

func main() {
	reg := rego.New(rego.Query("1==1"))
	preparedReg, err := reg.PrepareForEval(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	mechanismServers := map[string]networkservice.NetworkServiceServer{
		api_kernel.MECHANISM: kernel.NewServer(),
	}

	endpoint.NewServer("icmp-responder", &preparedReg, mechanisms.NewServer(mechanismServers))
}
