package probe

import (
	"context"
	"net/http"
	"sync"

	"github.com/wzshiming/schedialer"
)

var (
	defaultTransport     = http.DefaultTransport.(*http.Transport).Clone()
	defaultTransportPool = sync.Pool{
		New: func() interface{} {
			return defaultTransport.Clone()
		},
	}
)

func pingPong(ctx context.Context, uri string, dialer schedialer.Dialer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return err
	}
	transport := defaultTransportPool.Get().(*http.Transport)
	defer defaultTransportPool.Put(transport)
	transport.DialContext = dialer.DialContext
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.ContentLength != 0 {
		var tmp [1]byte
		_, err := resp.Body.Read(tmp[:])
		if err != nil {
			return err
		}
	}
	return nil
}
