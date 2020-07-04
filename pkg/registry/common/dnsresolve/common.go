package dnsresolve

import (
	"context"
	"net"
)

type Resolver interface {
	LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error)
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

func resolveDomain(ctx context.Context, domain string, r Resolver) *net.TCPAddr {
	_, records, err := r.LookupSRV(ctx, "", "tcp", domain)

	if err != nil {
		return nil
	}

	if len(records) == 0 {
		return nil
	}

	ips, err := r.LookupIPAddr(ctx, records[0].Target)

	if err != nil {
		return nil
	}

	if len(ips) == 0 {
		return nil
	}

	return &net.TCPAddr{Port: int(records[0].Port), IP: ips[0].IP}
}
