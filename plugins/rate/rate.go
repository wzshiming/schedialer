package rate

import (
	"context"

	"github.com/wzshiming/schedialer"
)

type Rate struct {
	score int
}

func NewRate() schedialer.Plugin {
	return &Rate{
		score: schedialer.MaxScore / 50,
	}
}

func (f *Rate) Name() string {
	return "Rate"
}

func (f *Rate) ComparisonScore(ctx context.Context, target *schedialer.Target, proxies []*schedialer.Proxy) ([]int, error) {
	var scores = make([]int, len(proxies))
	for i := range proxies {
		aver := proxies[i].Aver()
		maxAver := proxies[i].MaxAver()
		if maxAver > aver {
			scores[i] += int(float64(maxAver-aver) / float64(maxAver) * float64(f.score))
		}
	}

	return scores, nil
}
