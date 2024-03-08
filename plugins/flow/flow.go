package flow

import (
	"context"

	"github.com/wzshiming/schedialer"
)

type Flow struct {
	score int
}

func NewFlow() schedialer.Plugin {
	return &Flow{
		score: schedialer.MaxScore / 50,
	}
}

func (f *Flow) Name() string {
	return "Flow"
}

func (f *Flow) ComparisonScore(ctx context.Context, target *schedialer.Target, proxies []*schedialer.Proxy) ([]int, error) {
	var (
		totals = make([]uint64, 0, len(proxies))

		scores = make([]int, len(proxies))
	)

	var maxTotal uint64
	for i := range proxies {
		total := proxies[i].Total()
		totals = append(totals, total)
		if total > maxTotal {
			maxTotal = total
		}
	}

	for i, total := range totals {
		if maxTotal > total {
			scores[i] += int(float64(maxTotal-total) / float64(maxTotal) * float64(f.score))
		}
	}
	return scores, nil
}
