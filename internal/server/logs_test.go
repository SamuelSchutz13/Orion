package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SamuelSchutz13/orion-logger/internal/logs"
	"github.com/SamuelSchutz13/orion-logger/internal/session"
)

func send(t *testing.T, s *session.Session, method, u string, body any) *http.Response {
	t.Helper()
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	r, _ := http.NewRequest(method, u, bytes.NewReader(b))
	r.Header.Set("X-Orion-Token", s.Token)
	r.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// readLines lê eventos "lines" do SSE até achar want.
func readLines(t *testing.T, res *http.Response, want string) {
	t.Helper()
	done := make(chan bool, 1)
	go func() {
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), "data: ") {
				var ls []logs.Line
				json.Unmarshal([]byte(sc.Text()[6:]), &ls)
				for _, l := range ls {
					if l.Text == want {
						done <- true
						return
					}
				}
			}
		}
		done <- false
	}()
	select {
	case ok := <-done:
		if !ok {
			t.Fatalf("stream acabou sem %q", want)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout esperando %q", want)
	}
}

func TestLogsFromHub(t *testing.T) {
	hub := logs.NewHub(100)
	runs := NewRuns(hub)
	runs.Start("x", 1)
	hub.Append("run:1", "out", "antes")
	base, s := liveServer(t, Deps{Hub: hub, Runs: runs})
	res := get(t, s, base+"/api/logs?id=run:1")
	defer res.Body.Close()
	readLines(t, res, "antes")
	hub.Append("run:1", "out", "depois")
	readLines(t, res, "depois")
}

func TestLogFilesAddAndTail(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	p := filepath.Join(t.TempDir(), "app.log")
	os.WriteFile(p, []byte("ola\n"), 0o644)
	lf := NewLogFiles()
	base, s := liveServer(t, Deps{LogFiles: lf})

	if res := send(t, s, "POST", base+"/api/logfiles", map[string]string{"path": p + ".nao"}); res.StatusCode != 400 {
		t.Fatalf("arquivo inexistente: %d", res.StatusCode)
	}
	// arquivo fora da lista não pode ser lido
	if res := get(t, s, base+"/api/logs?id="+url.QueryEscape("file:"+p)); res.StatusCode != 404 {
		t.Fatalf("arquivo não adicionado: %d", res.StatusCode)
	}
	if res := send(t, s, "POST", base+"/api/logfiles", map[string]string{"path": p}); res.StatusCode != 200 {
		t.Fatalf("add: %d", res.StatusCode)
	}
	if len(session.LoadConfig().LogFiles) != 1 {
		t.Fatal("config.json não salvo")
	}
	res := get(t, s, base+"/api/logs?id="+url.QueryEscape("file:"+p))
	defer res.Body.Close()
	readLines(t, res, "ola")
}

func TestLogsUnknownKind404(t *testing.T) {
	base, s := liveServer(t, Deps{Hub: logs.NewHub(10)})
	if res := get(t, s, base+"/api/logs?id=proc:1:1"); res.StatusCode != 404 {
		t.Fatalf("%d", res.StatusCode)
	}
}

func TestLogsUnknownRun404(t *testing.T) {
	hub := logs.NewHub(10)
	base, s := liveServer(t, Deps{Hub: hub, Runs: NewRuns(hub)})
	if res := get(t, s, base+"/api/logs?id=run:99"); res.StatusCode != 404 {
		t.Fatalf("%d", res.StatusCode)
	}
	if hub.Has("run:99") {
		t.Fatal("stream vazio criado no hub")
	}
}
