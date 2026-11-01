package inventory

import (
	"context"
	"net"
	"os"
	"testing"
	"time"
)

// Teste contra a máquina real: o próprio processo de teste abre uma porta e deve aparecer com ela.
func TestProcSourceLive(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	s := NewProcSource()
	for i := 0; i < 2; i++ { // a 2ª rodada usa o cache
		start := time.Now()
		items, err := s.List(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("rodada %d: %d processos em %v", i+1, len(items), time.Since(start))
		found := false
		for _, it := range items {
			if it.PID == os.Getpid() {
				for _, p := range it.Ports {
					found = found || p == port
				}
			}
		}
		if !found {
			t.Fatalf("porta %d do PID %d não encontrada", port, os.Getpid())
		}
	}
}
