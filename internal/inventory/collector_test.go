package inventory

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeSource struct {
	name  string
	items []Item
	err   error
	block bool
}

func (f fakeSource) Name() string { return f.name }
func (f fakeSource) List(ctx context.Context) ([]Item, error) {
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.items, f.err
}

func TestFailingSourceDoesNotBreakOthers(t *testing.T) {
	c := NewCollector(time.Hour, nil, fakeSource{name: "bad", err: errors.New("boom")}, fakeSource{name: "good", items: []Item{{ID: "a"}}})
	snap := c.collect(context.Background())
	if len(snap.Items) != 1 || snap.Sources["bad"] != (SourceState{false, "boom"}) || !snap.Sources["good"].OK {
		t.Fatalf("%+v", snap)
	}
	if _, ok := c.Find("a"); !ok {
		t.Fatal("Find should see collected item")
	}
}

func TestSlowSourceTimesOut(t *testing.T) {
	c := NewCollector(time.Hour, nil, fakeSource{name: "slow", block: true})
	c.timeout = 100 * time.Millisecond // em produção são 5 s
	start := time.Now()
	snap := c.collect(context.Background())
	if time.Since(start) > time.Second || !strings.Contains(snap.Sources["slow"].Reason, "timeout") {
		t.Fatalf("%v %+v", time.Since(start), snap)
	}
}

func TestExtraItemsIncluded(t *testing.T) {
	c := NewCollector(time.Hour, func() []Item { return []Item{{ID: "run:1"}} })
	if snap := c.collect(context.Background()); len(snap.Items) != 1 || snap.Items[0].ID != "run:1" {
		t.Fatalf("%+v", snap)
	}
}

func TestSubscribeGetsLatest(t *testing.T) {
	c := NewCollector(10*time.Millisecond, nil, fakeSource{name: "good", items: []Item{{ID: "a"}}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	ch, unsub := c.Subscribe()
	defer unsub()
	select {
	case s := <-ch:
		if len(s.Items) != 1 {
			t.Fatalf("%+v", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no snapshot")
	}
}
