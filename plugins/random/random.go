package random

import (
	"context"
	"math/rand"
	"time"

	"github.com/wzshiming/schedialer"
)

type Random struct {
	src   rand.Source
	score int
}

type Option func(r *Random)

func WithScore(score int) Option {
	return func(r *Random) {
		r.score = score
	}
}

func NewRandom(opts ...Option) schedialer.Plugin {
	src := rand.NewSource(time.Now().UnixNano())
	r := &Random{
		src:   src,
		score: schedialer.MaxScore / 2,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Random) Name() string {
	return "Random"
}

func (r *Random) ComparisonScore(ctx context.Context, target *schedialer.Target, proxies []*schedialer.Proxy) ([]int, error) {
	scores := make([]int, len(proxies))
	scores[int(r.src.Int63())%len(proxies)] = r.score
	return scores, nil
}
