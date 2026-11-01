package server

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/SamuelSchutz13/orion-logger/internal/inventory"
	"github.com/SamuelSchutz13/orion-logger/internal/session"
)

type oneItem struct{}

func (oneItem) Name() string { return "fake" }
func (oneItem) List(context.Context) ([]inventory.Item, error) {
	return []inventory.Item{{ID: "proc:1:1", Name: "fake"}}, nil
}

// liveServer sobe um servidor HTTP real em 127.0.0.1 (o Host precisa bater com a porta).
func liveServer(t *testing.T, deps Deps) (base string, s *session.Session) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s, _ = session.New()
	s.Port = l.Addr().(*net.TCPAddr).Port
	srv := &http.Server{Handler: New(s, deps)}
	go srv.Serve(l)
	t.Cleanup(func() { srv.Close() })
	return "http://" + l.Addr().String(), s
}

func get(t *testing.T, s *session.Session, url string) *http.Response {
	t.Helper()
	r, _ := http.NewRequest("GET", url, nil)
	r.Header.Set("X-Orion-Token", s.Token)
	res, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestStreamSendsSnapshot(t *testing.T) {
	c := inventory.NewCollector(time.Hour, nil, oneItem{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	base, s := liveServer(t, Deps{Collector: c})
	res := get(t, s, base+"/api/stream")
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type %q", ct)
	}
	sc := bufio.NewScanner(res.Body)
	var got []string
	for sc.Scan() && len(got) < 2 {
		if sc.Text() != "" {
			got = append(got, sc.Text())
		}
	}
	if got[0] != "event: snapshot" || !strings.Contains(got[1], `"items":[{"id":"proc:1:1"`) {
		t.Fatalf("%q", got)
	}
}
