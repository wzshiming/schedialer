package schedialer

import (
	"context"
	"fmt"
	"net"
	"sort"
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type Proxy struct {
	Name   string
	IP     net.IP
	Port   int
	Dialer Dialer

	Aver    func() uint64
	MaxAver func() uint64

	Total func() uint64
}

func (p *Proxy) String() string {
	return p.Name
}

type Target struct {
	Address string
	IPs     []net.IP
	Port    int
}

type Plugin interface {
	Name() string
}

type AddAndDelPlugin interface {
	Plugin
	OnAdd(ctx context.Context, proxy *Proxy) error
	OnDel(ctx context.Context, proxy *Proxy) error
}

type ScorePlugin interface {
	Plugin
	Score(ctx context.Context, target *Target, proxy *Proxy) (int, error)
}

type ComparisonScorePlugin interface {
	Plugin
	ComparisonScore(ctx context.Context, target *Target, proxies []*Proxy) ([]int, error)
}

type FilterPlugin interface {
	Plugin
	Filter(ctx context.Context, target *Target, proxy *Proxy) bool
}

type Feedback struct {
	Successful bool
	Error      error
}

type FeedbackPlugin interface {
	Plugin
	Feedback(ctx context.Context, target *Target, proxy *Proxy, feedback *Feedback)
}

type Plugins struct {
	AddAndDelPlugins       []AddAndDelPlugin
	FilterPlugins          []FilterPlugin
	ComparisonScorePlugins []ComparisonScorePlugin
	ScorePlugins           []ScorePlugin
	FeedbackPlugins        []FeedbackPlugin
	Proxies                map[string]*Proxy
}

func NewPlugins(plugins ...Plugin) *Plugins {
	m := &Plugins{
		Proxies: map[string]*Proxy{},
	}
	m.Register(plugins...)
	return m
}

func (m *Plugins) Register(plugins ...Plugin) {
	for _, plugin := range plugins {
		m.register(plugin)
	}
}

func (m *Plugins) register(plugin Plugin) {
	if p, ok := plugin.(AddAndDelPlugin); ok {
		m.AddAndDelPlugins = append(m.AddAndDelPlugins, p)
	}
	if p, ok := plugin.(FilterPlugin); ok {
		m.FilterPlugins = append(m.FilterPlugins, p)
	}
	if p, ok := plugin.(ComparisonScorePlugin); ok {
		m.ComparisonScorePlugins = append(m.ComparisonScorePlugins, p)
	}
	if p, ok := plugin.(ScorePlugin); ok {
		m.ScorePlugins = append(m.ScorePlugins, p)
	}
	if p, ok := plugin.(FeedbackPlugin); ok {
		m.FeedbackPlugins = append(m.FeedbackPlugins, p)
	}
}

func (m *Plugins) AddProxy(ctx context.Context, proxy *Proxy) error {
	uniq := proxy.String()
	_, ok := m.Proxies[uniq]
	if ok {
		return nil
	}
	for _, plugin := range m.AddAndDelPlugins {
		err := plugin.OnAdd(ctx, proxy)
		if err != nil {
			return err
		}
	}
	m.Proxies[uniq] = proxy
	return nil
}

func (m *Plugins) DelProxy(ctx context.Context, proxy *Proxy) error {
	uniq := proxy.String()
	proxy, ok := m.Proxies[uniq]
	if !ok {
		return nil
	}
	for _, plugin := range m.AddAndDelPlugins {
		err := plugin.OnDel(ctx, proxy)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Plugins) Ranking(ctx context.Context, target *Target) ([]*Proxy, error) {
	proxies, scores, err := m.match(ctx, target)
	if err != nil {
		return nil, err
	}
	sort.Sort(&proxyScoreSorter{
		proxies: proxies,
		scores:  scores,
	})
	return proxies, nil
}

func (m *Plugins) match(ctx context.Context, target *Target) ([]*Proxy, []int, error) {
	proxies := make([]*Proxy, 0, len(m.Proxies))
	scores := make([]int, 0, len(m.Proxies))
loop:
	for _, proxy := range m.Proxies {
		for _, plugin := range m.FilterPlugins {
			if !plugin.Filter(ctx, target, proxy) {
				continue loop
			}
		}
		proxies = append(proxies, proxy)

		score := 0
		for _, plugin := range m.ScorePlugins {
			s, err := plugin.Score(ctx, target, proxy)
			if err != nil {
				return nil, nil, err
			}
			score += s
		}
		scores = append(scores, score)
	}

	if len(proxies) == 0 {
		return nil, nil, fmt.Errorf("not match")
	}

	for _, plugin := range m.ComparisonScorePlugins {
		s, err := plugin.ComparisonScore(ctx, target, proxies)
		if err != nil {
			return nil, nil, err
		}
		scoresAdd(&scores, s)
	}

	return proxies, scores, nil
}

func (m *Plugins) Feedback(ctx context.Context, target *Target, proxy *Proxy, feedback *Feedback) {
	for _, plugin := range m.FeedbackPlugins {
		plugin.Feedback(ctx, target, proxy, feedback)
	}
}

func scoresAdd(p *[]int, e []int) {
	for i := range *p {
		(*p)[i] += e[i]
	}
}

type proxyScoreSorter struct {
	proxies []*Proxy
	scores  []int
}

func (p *proxyScoreSorter) Len() int {
	return len(p.scores)
}

func (p *proxyScoreSorter) Less(i, j int) bool {
	return p.scores[i] > p.scores[j]
}

func (p *proxyScoreSorter) Swap(i, j int) {
	p.scores[i], p.scores[j] = p.scores[j], p.scores[i]
	p.proxies[i], p.proxies[j] = p.proxies[j], p.proxies[i]
}
