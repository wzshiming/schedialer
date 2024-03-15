package roundrobin

import (
	"context"
	"sync/atomic"

	"github.com/wzshiming/schedialer"
)

type RoundRobin struct {
	index uint64
	score int
}

func NewRoundRobin(score int) schedialer.Plugin {
	return &RoundRobin{
		score: score,
	}
}

func (r *RoundRobin) Name() string {
	return "RoundRobin"
}

func (r *RoundRobin) ComparisonScore(ctx context.Context, target *schedialer.Target, proxies []*schedialer.Proxy) ([]int, error) {
	n := atomic.AddUint64(&r.index, 1)
	n = (n - 1) % uint64(len(proxies))

	scores := make([]int, len(proxies))
	scores[n] = r.score
	return scores, nil
}
