package random

import (
	"context"
	"math/rand"
	"time"

	"github.com/wzshiming/schedialer"
)

type Random struct {
	src rand.Source
}

func NewRandom() schedialer.Plugin {
	src := rand.NewSource(time.Now().UnixNano())
	return &Random{
		src: src,
	}
}

func (r *Random) Name() string {
	return "Random"
}

func (r *Random) ComparisonScore(ctx context.Context, target *schedialer.Target, proxies []*schedialer.Proxy) ([]int, error) {
	scores := make([]int, len(proxies))
	scores[int(r.src.Int63())%len(proxies)] = schedialer.MaxScore / 2
	return scores, nil
}
