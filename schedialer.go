package schedialer

import (
	"context"
	"net"
)

func NewSchedialer(plugins *Plugins) *Schedialer {
	return &Schedialer{
		Plugins:  plugins,
		Resolver: net.DefaultResolver,
	}
}

type Schedialer struct {
	Plugins  *Plugins
	Resolver *net.Resolver
}

func (s *Schedialer) Match(ctx context.Context, network, address string) (*Proxy, error) {
	resolver := s.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	var ips []net.IP
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err = resolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
	} else {
		ips = []net.IP{ip}
	}

	port, err := resolver.LookupPort(ctx, network, portStr)
	if err != nil {
		return nil, err
	}
	target := Target{
		Address: address,
		IPs:     ips,
		Port:    port,
	}
	return s.Plugins.Match(ctx, &target)
}

func (s *Schedialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	proxy, err := s.Match(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return proxy.Dialer.DialContext(ctx, network, address)
}
