package backoff

import (
	"context"
	"sync"
	"time"

	"github.com/wzshiming/schedialer"
)

type Backoff struct {
	mutChecks sync.RWMutex
	checks    map[string]*info
}

type info struct {
	proxy *schedialer.Proxy

	mut sync.RWMutex

	failCont     int
	failLastTime time.Time
}

func NewBackoff() schedialer.Plugin {
	return &Backoff{
		checks: map[string]*info{},
	}
}

func (p *Backoff) Name() string {
	return "Backoff"
}

func (p *Backoff) OnAdd(ctx context.Context, proxy *schedialer.Proxy) error {
	n := &info{
		proxy: proxy,
	}

	p.mutChecks.Lock()
	defer p.mutChecks.Unlock()
	p.checks[proxy.String()] = n
	return nil
}

func (p *Backoff) OnDel(ctx context.Context, proxy *schedialer.Proxy) error {
	p.mutChecks.Lock()
	defer p.mutChecks.Unlock()
	delete(p.checks, proxy.String())
	return nil
}

func (p *Backoff) Filter(ctx context.Context, target *schedialer.Target, proxy *schedialer.Proxy) bool {
	p.mutChecks.RLock()
	defer p.mutChecks.RUnlock()

	info := p.checks[proxy.String()]
	info.mut.RLock()
	defer info.mut.RUnlock()

	if info.failCont == 0 {
		return true
	}

	if info.failLastTime.Add(time.Second << uint(info.failCont)).Before(time.Now()) {
		return true
	}
	return false
}

func (p *Backoff) Feedback(ctx context.Context, target *schedialer.Target, proxy *schedialer.Proxy, feedback *schedialer.Feedback) {
	p.mutChecks.RLock()
	defer p.mutChecks.RUnlock()

	info := p.checks[proxy.String()]
	info.mut.Lock()
	defer info.mut.Unlock()

	if feedback.Successful {
		info.failCont = 0
	} else {
		info.failCont++
		info.failLastTime = time.Now()
	}
}
