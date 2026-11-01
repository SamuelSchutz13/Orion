package runner

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/SamuelSchutz13/orion-logger/internal/inventory"
	"github.com/SamuelSchutz13/orion-logger/internal/logs"
	"github.com/SamuelSchutz13/orion-logger/internal/server"
	"github.com/SamuelSchutz13/orion-logger/internal/session"
)

type env struct {
	hub  *logs.Hub
	col  *inventory.Collector
	runs *server.Runs
}

// serve sobe um servidor real e grava o session.json apontando para ele.
func serve(t *testing.T) env {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s, _ := session.New()
	s.Port = l.Addr().(*net.TCPAddr).Port
	hub := logs.NewHub(100)
	runs := server.NewRuns(hub)
	col := inventory.NewCollector(20*time.Millisecond, runs.Merge)
	ctx, cancel := context.WithCancel(context.Background())
	go col.Run(ctx)
	srv := &http.Server{Handler: server.New(s, server.Deps{Collector: col, Hub: hub, Runs: runs})}
	go srv.Serve(l)
	t.Cleanup(func() { cancel(); srv.Close() })
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	return env{hub, col, runs}
}

func texts(h *logs.Hub, id string) map[string]string {
	back, _, cancel := h.Subscribe(id)
	defer cancel()
	m := map[string]string{}
	for _, l := range back {
		m[l.Text] = l.Stream
	}
	return m
}

func TestRunExitCode(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	serve(t)
	if code := Run([]string{"cmd", "/c", "exit 3"}); code != 3 {
		t.Fatalf("code %d", code)
	}
}

func TestRunForwardsLines(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	e := serve(t)
	if code := Run([]string{"cmd", "/c", "echo oi& echo erro 1>&2"}); code != 0 {
		t.Fatalf("code %d", code)
	}
	got := texts(e.hub, "run:1")
	if got["oi"] != "out" || got["erro "] != "err" {
		t.Fatalf("%q", got)
	}
	time.Sleep(100 * time.Millisecond) // próxima rodada do coletor
	it, ok := e.col.Find("run:1")
	if !ok || it.Status != inventory.StatusDown || it.Detail["exitCode"] != "0" {
		t.Fatalf("%+v", it)
	}
}

func TestRunStartsServerWhenStale(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	stale, _ := session.New()
	stale.Port = 1 // porta morta
	stale.Save()
	var e env
	started := false
	old := startServer
	startServer = func() error { started = true; e = serve(t); return nil }
	defer func() { startServer = old }()
	if code := Run([]string{"cmd", "/c", "echo depois"}); code != 0 {
		t.Fatalf("code %d", code)
	}
	if !started || texts(e.hub, "run:1")["depois"] != "out" {
		t.Fatalf("started=%v", started)
	}
}

func TestRunWithoutServerStillRuns(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	old := startServer
	startServer = func() error { return nil } // servidor nunca aparece
	defer func() { startServer = old }()
	waitServer = 200 * time.Millisecond
	if code := Run([]string{"cmd", "/c", "exit 5"}); code != 5 {
		t.Fatalf("code %d", code)
	}
}

func TestRunMissingCommand(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	serve(t)
	if code := Run([]string{"comando-que-nao-existe-123"}); code == 0 {
		t.Fatal("deveria falhar")
	}
}

func TestExistingFindsLiveServer(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	if _, ok := Existing(); ok {
		t.Fatal("sem session.json não há servidor")
	}
	serve(t)
	s, ok := Existing()
	if !ok || s.Port == 0 || s.Token == "" {
		t.Fatalf("servidor vivo não encontrado: %+v", s)
	}
}

func TestExistingIgnoresStaleSession(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	stale, _ := session.New()
	stale.Port = 1
	stale.Save()
	if _, ok := Existing(); ok {
		t.Fatal("session.json velho não deve contar como servidor vivo")
	}
}
