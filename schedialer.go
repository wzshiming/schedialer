package schedialer

import (
	"context"
	"fmt"
	"net"
	"time"
)

func NewSchedialer(plugins *Plugins) *Schedialer {
	return &Schedialer{
		Plugins:  plugins,
		Resolver: net.DefaultResolver,
		Period:   time.Second,
	}
}

type Schedialer struct {
	Plugins  *Plugins
	Resolver *net.Resolver
	Period   time.Duration
}

func (s *Schedialer) Ranking(ctx context.Context, network, address string) ([]*Proxy, error) {
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
	return s.Plugins.Ranking(ctx, &target)
}

func (s *Schedialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	proxies, err := s.Ranking(ctx, network, address)
	if err != nil {
		return nil, err
	}

	switch len(proxies) {
	case 0:
		return nil, fmt.Errorf("no proxy available")
	case 1:
		return proxies[0].Dialer.DialContext(ctx, network, address)
	}
	return fallbackDial(proxies, ctx, network, address)
}

func fallbackDial(proxies []*Proxy, ctx context.Context, network, address string) (net.Conn, error) {
	var err error
	for _, proxy := range proxies {
		conn, e := proxy.Dialer.DialContext(ctx, network, address)
		if e != nil {
			if err == nil {
				err = e
			}
			continue
		}
		return conn, nil
	}
	return nil, err
}
