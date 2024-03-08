package probe

import (
	"context"
	"sync"
	"time"

	"github.com/wzshiming/schedialer"
)

type Probe struct {
	uri     string
	timeout time.Duration
	period  time.Duration
	queue   chan *info
	refresh chan struct{}
	score   int

	mutChecks sync.RWMutex
	checks    map[string]*info

	mutLastUse sync.RWMutex
	lastUse    time.Time

	once sync.Once
}

type info struct {
	proxy *schedialer.Proxy

	mut sync.RWMutex

	response   bool
	duration   time.Duration
	err        error
	lastUpdate time.Time
}

func NewProbe(score int, uri string) schedialer.Plugin {
	return &Probe{
		uri:     uri,
		queue:   make(chan *info, 1),
		checks:  map[string]*info{},
		timeout: 5 * time.Second,
		period:  30 * time.Second,
		refresh: make(chan struct{}, 1),
		score:   score,
	}
}

func (p *Probe) Name() string {
	return "Probe"
}

func (p *Probe) start() {
	ctx := context.Background()

	go func() {
		for check := range p.queue {
			pass := make(chan struct{})
			go func() {
				p.task(ctx, check)
				close(pass)
			}()
			select {
			case <-pass:
			case <-time.After(time.Second):
			}
		}
	}()

	go func() {
		last := time.Now()
		period := p.period
		for {
			next := last.Add(period)
			d := next.Sub(time.Now())
			if d > 0 {
				select {
				case <-time.After(d):
				case <-p.refresh:
					period = p.period
					continue
				}
			}
			p.toStart()
			last = time.Now()
			period <<= 1
		}
	}()
	return
}

func (p *Probe) toStart() {
	p.mutChecks.RLock()
	defer p.mutChecks.RUnlock()
	for _, info := range p.checks {
		info.mut.RLock()
		pass := info.lastUpdate.Add(p.period).Before(time.Now())
		info.mut.RUnlock()
		if pass {
			p.queue <- info
		}
	}
	return
}

func (p *Probe) task(ctx context.Context, info *info) {
	startAt := time.Now()
	info.mut.RLock()
	pass := info.lastUpdate.Add(p.period).Before(startAt)
	info.mut.RUnlock()
	if !pass {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	err := pingPong(ctx, p.uri, info.proxy.Dialer)
	cancel()

	duration := time.Since(startAt)

	info.mut.Lock()
	defer info.mut.Unlock()
	if err != nil {
		info.response = false
		info.duration = 0
		info.err = err
	} else {
		info.response = true
		info.duration = duration
		info.err = nil
	}
	info.lastUpdate = time.Now()
}

func (p *Probe) OnAdd(ctx context.Context, proxy *schedialer.Proxy) error {
	p.once.Do(p.start)
	n := &info{
		proxy: proxy,
	}
	p.queue <- n

	p.mutChecks.Lock()
	defer p.mutChecks.Unlock()
	p.checks[proxy.String()] = n
	return nil
}

func (p *Probe) OnDel(ctx context.Context, proxy *schedialer.Proxy) error {
	p.mutChecks.Lock()
	defer p.mutChecks.Unlock()
	delete(p.checks, proxy.String())
	return nil
}

func (p *Probe) Filter(ctx context.Context, target *schedialer.Target, proxy *schedialer.Proxy) bool {
	p.mutChecks.RLock()
	defer p.mutChecks.RUnlock()
	info := p.checks[proxy.String()]
	info.mut.RLock()
	defer info.mut.RUnlock()
	return info.response
}

func (p *Probe) ComparisonScore(ctx context.Context, target *schedialer.Target, proxies []*schedialer.Proxy) ([]int, error) {
	select {
	case p.refresh <- struct{}{}:
	default:
	}

	p.mutLastUse.Lock()
	p.lastUse = time.Now()
	p.mutLastUse.Unlock()

	durations := make([]time.Duration, 0, len(proxies))
	p.mutChecks.RLock()
	for _, proxy := range proxies {
		info := p.checks[proxy.String()]
		info.mut.RLock()
		durations = append(durations, info.duration)
		info.mut.RUnlock()
	}
	p.mutChecks.RUnlock()

	min := time.Duration(1<<63 - 1)
	for _, duration := range durations {
		if duration < min {
			min = duration
		}
	}

	scores := make([]int, 0, len(proxies))
	for _, duration := range durations {
		score := 0
		if duration == min {
			score = p.score
		} else {
			score = int(float64(p.score) * float64(min) / float64(duration))
		}
		scores = append(scores, score)
	}
	return scores, nil
}
