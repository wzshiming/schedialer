package schedialer

import (
	"context"
	"fmt"
	"net"
	"time"
	"errors"
)

func NewSchedialer(plugins *Plugins) *Schedialer {
	return &Schedialer{
		Plugins:     plugins,
		Resolver:    net.DefaultResolver,
		DialTimeout: 10 * time.Second,
	}
}

type Schedialer struct {
	Plugins     *Plugins
	Resolver    *net.Resolver
	DialTimeout time.Duration
}

func (s *Schedialer) getTarget(ctx context.Context, network, address string) (*Target, error) {
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
	return &Target{
		Address: address,
		IPs:     ips,
		Port:    port,
	}, nil
}

func (s *Schedialer) Ranking(ctx context.Context, network, address string) ([]*Proxy, error) {
	target, err := s.getTarget(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return s.Plugins.Ranking(ctx, target)
}

func (s *Schedialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	proxies, err := s.Ranking(ctx, network, address)
	if err != nil {
		return nil, err
	}

	switch len(proxies) {
	case 0:
		return nil, fmt.Errorf("no proxy available")
	}
	return s.fallbackDial(proxies, ctx, network, address)
}

func (s *Schedialer) fallbackDial(proxies []*Proxy, ctx context.Context, network, address string) (net.Conn, error) {
	target, err := s.getTarget(ctx, network, address)
	if err != nil {
		return nil, err
	}

	var errs []error
	for _, proxy := range proxies {
		dialTimeout, cancel := context.WithTimeout(ctx, s.DialTimeout)
		conn, err := proxy.Dialer.DialContext(dialTimeout, network, address)
		cancel()
		if err == nil {
			s.Plugins.Feedback(ctx, target, proxy, &Feedback{
				Successful: true,
			})
			return conn, nil
		}

		s.Plugins.Feedback(ctx, target, proxy, &Feedback{
			Error: err,
		})

		errs = append(errs, err)
	}
	return nil, errors.Join(errs...)
}
