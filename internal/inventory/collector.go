package inventory

import (
	"context"
	"sync"
	"time"
)

// Collector consulta as fontes em paralelo a cada intervalo e entrega o snapshot aos assinantes.
type Collector struct {
	interval time.Duration
	timeout  time.Duration
	extra    func([]Item) []Item // ajusta a lista final (itens do orion run e arquivos .log)
	srcs     []Source
	Machine  func(context.Context) *Machine // opcional

	mu     sync.Mutex
	latest Snapshot
	ready  bool // já houve uma coleta
	byID   map[string]Item
	subs   map[chan Snapshot]struct{}
	kick   chan struct{}
}

func NewCollector(interval time.Duration, extra func([]Item) []Item, srcs ...Source) *Collector {
	return &Collector{
		interval: interval, timeout: 5 * time.Second, extra: extra, srcs: srcs,
		latest: Snapshot{Items: []Item{}, Sources: map[string]SourceState{}},
		byID:   map[string]Item{}, subs: map[chan Snapshot]struct{}{}, kick: make(chan struct{}, 1),
	}
}

func (c *Collector) Run(ctx context.Context) {
	t := time.NewTicker(c.interval)
	defer t.Stop()
	for {
		c.publish(c.collect(ctx))
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-c.kick:
		}
	}
}

// Refresh antecipa a próxima rodada (por exemplo, logo depois de uma ação).
func (c *Collector) Refresh() {
	select {
	case c.kick <- struct{}{}:
	default:
	}
}

type result struct {
	items []Item
	err   error
}

func (c *Collector) collect(ctx context.Context) Snapshot {
	snap := Snapshot{Items: []Item{}, Sources: map[string]SourceState{}}
	results := make([]result, len(c.srcs))
	var wg sync.WaitGroup
	for i, s := range c.srcs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sctx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()
			done := make(chan result, 1)
			// A fonte pode ignorar o ctx; o select garante que a rodada não trava por causa dela.
			go func() { it, err := s.List(sctx); done <- result{it, err} }()
			select {
			case r := <-done:
				if r.err != nil && sctx.Err() != nil {
					r.err = context.DeadlineExceeded
				}
				results[i] = r
			case <-sctx.Done():
				results[i] = result{err: context.DeadlineExceeded}
			}
		}()
	}
	var machine *Machine
	if c.Machine != nil {
		wg.Add(1)
		go func() { defer wg.Done(); machine = c.Machine(ctx) }()
	}
	wg.Wait()
	snap.Machine = machine
	for i, s := range c.srcs {
		r := results[i]
		switch {
		case r.err == context.DeadlineExceeded:
			snap.Sources[s.Name()] = SourceState{false, "timeout: a fonte demorou mais de " + c.timeout.String()}
		case r.err != nil:
			snap.Sources[s.Name()] = SourceState{false, r.err.Error()}
		default:
			snap.Sources[s.Name()] = SourceState{OK: true}
			snap.Items = append(snap.Items, r.items...)
		}
	}
	if c.extra != nil {
		snap.Items = c.extra(snap.Items)
	}
	byID := make(map[string]Item, len(snap.Items))
	for _, it := range snap.Items {
		byID[it.ID] = it
	}
	c.mu.Lock()
	c.latest, c.byID, c.ready = snap, byID, true
	c.mu.Unlock()
	return snap
}

func (c *Collector) publish(s Snapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for ch := range c.subs {
		// assinante lento: descarta o snapshot antigo e fica só com o mais novo
		select {
		case <-ch:
		default:
		}
		ch <- s
	}
}

// Subscribe já entrega o último snapshot (se houver) no canal.
func (c *Collector) Subscribe() (<-chan Snapshot, func()) {
	ch := make(chan Snapshot, 1)
	c.mu.Lock()
	c.subs[ch] = struct{}{}
	if c.ready {
		ch <- c.latest
	}
	c.mu.Unlock()
	return ch, func() {
		c.mu.Lock()
		delete(c.subs, ch)
		c.mu.Unlock()
	}
}

func (c *Collector) Latest() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.latest
}

func (c *Collector) Find(id string) (Item, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.byID[id]
	return it, ok
}
