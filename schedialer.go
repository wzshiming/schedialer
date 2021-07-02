package schedialer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
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

	return quickDial(ctx, network, address, s.Period, proxies)
}

func quickDial(ctx context.Context, network, address string, period time.Duration, proxies []*Proxy) (net.Conn, error) {
	chconns := make(chan net.Conn)
	cherrs := make(chan error)
	go func() {
		quickDialChan(ctx, network, address, period, proxies, chconns, cherrs)
		close(chconns)
		close(cherrs)

		for conn := range chconns {
			conn.Close()
		}
	}()

	conn, ok := <-chconns
	if ok {
		return conn, nil
	}
	err := <-cherrs
	if err == nil {
		err = context.Canceled
	}
	return nil, err
}

func quickDialChan(ctx context.Context, network, address string, period time.Duration, proxies []*Proxy, chconns chan net.Conn, cherrs chan error) {
	var wg sync.WaitGroup
	timer := time.NewTicker(period)
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		timer.Stop()
		wg.Wait()
	}()

	for _, proxy := range proxies {
		wg.Add(1)
		go func(proxy *Proxy) {
			defer wg.Done()
			conn, err := proxy.Dialer.DialContext(ctx, network, address)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				select {
				default:
				case cherrs <- err:
				}
				return
			}
			chconns <- conn
		}(proxy)
		select {
		case <-timer.C:
		case <-ctx.Done():
			return
		}
	}
}
