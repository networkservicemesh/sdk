package sandbox

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"

	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
)

type FakeDNSResolver struct {
	sync.Mutex
	ports map[string]string
}

func (f *FakeDNSResolver) LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
	f.Lock()
	defer f.Unlock()
	if f.ports == nil {
		f.ports = map[string]string{}
	}
	if v, ok := f.ports[name]; ok {
		i, err := strconv.Atoi(v)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("_%v._%v.%v", service, proto, name), []*net.SRV{{
			Port:   uint16(i),
			Target: name,
		}}, nil
	}
	return "", nil, errors.New("not found")
}

func (f *FakeDNSResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	f.Lock()
	defer f.Unlock()
	if f.ports == nil {
		f.ports = map[string]string{}
	}
	if _, ok := f.ports[host]; ok {
		return []net.IPAddr{{
			IP: net.ParseIP("127.0.0.1"),
		}}, nil
	}
	return nil, errors.New("not found")
}

func (f *FakeDNSResolver) Register(name string, url *url.URL) error {
	f.Lock()
	defer f.Unlock()
	if f.ports == nil {
		f.ports = map[string]string{}
	}
	key := fmt.Sprintf("%v.%v", dnsresolve.NSMRegistryService, name)
	var err error
	_, f.ports[key], err = net.SplitHostPort(url.Host)
	return err
}

var _ dnsresolve.Resolver = (*FakeDNSResolver)(nil)
