package logs

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

func TestHubCapsBacklog(t *testing.T) {
	h := NewHub(5000)
	for i := 0; i < 6000; i++ {
		h.Append("run:1", "out", strconv.Itoa(i))
	}
	back, _, cancel := h.Subscribe("run:1")
	defer cancel()
	if len(back) != 5000 || back[0].Text != "1000" || back[4999].Text != "5999" {
		t.Fatalf("len %d first %q", len(back), back[0].Text)
	}
}

func TestHubInvalidUTF8(t *testing.T) {
	h := NewHub(10)
	h.Append("run:1", "out", "ol\xe1")
	back, _, cancel := h.Subscribe("run:1")
	defer cancel()
	if !utf8.ValidString(back[0].Text) || !strings.Contains(back[0].Text, "�") {
		t.Fatalf("%q", back[0].Text)
	}
}

func TestHubSubscriberGetsNew(t *testing.T) {
	h := NewHub(10)
	_, ch, cancel := h.Subscribe("run:1")
	defer cancel()
	h.Append("run:1", "err", "oi")
	select {
	case l := <-ch:
		if l.Text != "oi" || l.Stream != "err" || l.Seq == 0 {
			t.Fatalf("%+v", l)
		}
	case <-time.After(time.Second):
		t.Fatal("nada chegou")
	}
}

func TestHubSlowSubscriberDoesNotBlock(t *testing.T) {
	h := NewHub(100)
	_, _, cancel := h.Subscribe("run:1") // nunca lê
	defer cancel()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 5000; i++ {
			h.Append("run:1", "out", "x")
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Append bloqueou com assinante lento")
	}
}

// collector seguro para usar como Emit nos testes de tail
type got struct {
	mu    sync.Mutex
	lines []string
}

func (g *got) emit(stream, text string) { g.mu.Lock(); g.lines = append(g.lines, stream+":"+text); g.mu.Unlock() }
func (g *got) waitFor(t *testing.T, want string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		g.mu.Lock()
		for _, l := range g.lines {
			if strings.Contains(l, want) {
				g.mu.Unlock()
				return
			}
		}
		g.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("não recebeu %q; recebeu %q", want, g.lines)
}

func startTail(t *testing.T, path string) *got {
	g := &got{}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go Tail(ctx, path, g.emit)
	return g
}

func TestTailFollowsAppend(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.log")
	os.WriteFile(p, []byte("um\ndois\n"), 0o644)
	g := startTail(t, p)
	g.waitFor(t, "out:dois")
	f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString("tres\n")
	f.Close()
	g.waitFor(t, "out:um")
	g.waitFor(t, "out:tres")
}

func TestTailTruncation(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.log")
	os.WriteFile(p, []byte("linha antiga bem comprida\n"), 0o644)
	g := startTail(t, p)
	g.waitFor(t, "antiga")
	os.WriteFile(p, []byte("novo\n"), 0o644)
	g.waitFor(t, "out:novo")
}

func TestTailMissingFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "depois.log")
	g := startTail(t, p)
	g.waitFor(t, "sys:arquivo não encontrado")
	os.WriteFile(p, []byte("apareceu\n"), 0o644)
	g.waitFor(t, "out:apareceu")
}

func TestTailStartsAtLast500(t *testing.T) {
	p := filepath.Join(t.TempDir(), "big.log")
	var b strings.Builder
	for i := 0; i < 2000; i++ {
		b.WriteString("l" + strconv.Itoa(i) + "\n")
	}
	os.WriteFile(p, []byte(b.String()), 0o644)
	g := startTail(t, p)
	g.waitFor(t, "out:l1999")
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.lines) != 500 || g.lines[0] != "out:l1500" {
		t.Fatalf("len %d first %q", len(g.lines), g.lines[0])
	}
}
