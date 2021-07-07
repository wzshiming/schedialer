package test

import (
	"bufio"
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wzshiming/schedialer"
	"github.com/wzshiming/schedialer/plugins/probe"
)

func TestAll(t *testing.T) {
	svc := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		w := bufio.NewWriter(rw)
		for i := 0; i != 10000; i++ {
			w.Write([]byte("x"))
		}
		w.Flush()
	}))
	mgr := schedialer.NewPlugins(
		probe.NewProbe(svc.URL),
	)
	sche := schedialer.NewSchedialer(mgr)
	ctx := context.Background()

	go func() {
		mgr.AddProxy(ctx, &schedialer.Proxy{
			IP:     net.IPv4(127, 0, 0, 1),
			Port:   8000,
			Dialer: newThrottlingDialer(nil, time.Millisecond, 100),
		})
		mgr.AddProxy(ctx, &schedialer.Proxy{
			IP:     net.IPv4(127, 0, 0, 1),
			Port:   8001,
			Dialer: newThrottlingDialer(nil, time.Second, 1),
		})
		mgr.AddProxy(ctx, &schedialer.Proxy{
			IP:     net.IPv4(127, 0, 0, 1),
			Port:   8002,
			Dialer: newThrottlingDialer(nil, time.Second, 1),
		})
		mgr.AddProxy(ctx, &schedialer.Proxy{
			IP:     net.IPv4(127, 0, 0, 1),
			Port:   8003,
			Dialer: newThrottlingDialer(nil, time.Millisecond, 900),
		})
		mgr.AddProxy(ctx, &schedialer.Proxy{
			IP:     net.IPv4(127, 0, 0, 1),
			Port:   8004,
			Dialer: newThrottlingDialer(nil, time.Millisecond, 10),
		})
		mgr.AddProxy(ctx, &schedialer.Proxy{
			IP:     net.IPv4(127, 0, 0, 1),
			Port:   8005,
			Dialer: newThrottlingDialer(nil, time.Millisecond, 10),
		})
		mgr.AddProxy(ctx, &schedialer.Proxy{
			IP:     net.IPv4(127, 0, 0, 1),
			Port:   8006,
			Dialer: newThrottlingDialer(nil, time.Millisecond, 10),
		})
		mgr.AddProxy(ctx, &schedialer.Proxy{
			IP:     net.IPv4(127, 0, 0, 1),
			Port:   8007,
			Dialer: newThrottlingDialer(nil, time.Millisecond, 1000),
		})
	}()

	for {

		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.ForceAttemptHTTP2 = false
		transport.DialContext = sche.DialContext
		//transport.DisableKeepAlives = true
		cli := http.Client{
			Transport: transport,
		}

		req, err := http.NewRequest(http.MethodGet, svc.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		startAt := time.Now()
		resp, err := cli.Do(req)
		if err != nil {
			t.Error(err)
			continue
		}
		io.Copy(io.Discard, resp.Body)

		t.Log(time.Since(startAt), resp.ContentLength)
		resp.Body.Close()

		time.Sleep(time.Second)
	}
}

func newThrottlingDialer(dialer schedialer.Dialer, period time.Duration, bufSize int) schedialer.Dialer {
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	return &throttlingDialer{
		dialer:  dialer,
		period:  period,
		bufSize: bufSize,
	}
}

type throttlingDialer struct {
	dialer  schedialer.Dialer
	bufSize int
	period  time.Duration
}

func (t throttlingDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	log.Println("Conn", t.period, t.bufSize)
	conn, err := t.dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &throttlingConn{
		Conn:   conn,
		dialer: t,
	}, nil
}

type throttlingConn struct {
	net.Conn
	dialer throttlingDialer
}

func (t *throttlingConn) Read(b []byte) (n int, err error) {
	if len(b) > t.dialer.bufSize {
		b = b[:t.dialer.bufSize]
	}
	time.Sleep(t.dialer.period)
	return t.Conn.Read(b)
}

func (t *throttlingConn) Write(b []byte) (n int, err error) {
	if len(b) > t.dialer.bufSize {
		b = b[:t.dialer.bufSize]
	}
	time.Sleep(t.dialer.period)
	return t.Conn.Write(b)
}
