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

func NewRandom() schedialer.Plugin {
	src := rand.NewSource(time.Now().UnixNano())
	return &Random{
		src:   src,
		score: schedialer.MaxScore / 2,
	}
}

func (r *Random) Name() string {
	return "Random"
}

func (r *Random) ComparisonScore(ctx context.Context, target *schedialer.Target, proxies []*schedialer.Proxy) ([]int, error) {
	scores := make([]int, len(proxies))
	scores[int(r.src.Int63())%len(proxies)] = r.score
	return scores, nil
}
