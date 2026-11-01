// Package runner implementa `orion run <comando>`: roda o comando como filho deste terminal
// e copia stdout/stderr para o painel.
package runner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/SamuelSchutz13/orion-logger/internal/logs"
	"github.com/SamuelSchutz13/orion-logger/internal/session"
)

// Variáveis para os testes trocarem.
var (
	startServer = startDetached
	waitServer  = 5 * time.Second
)

type api struct {
	base, token string
	http        *http.Client
}

func (a *api) post(path string, body, out any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", a.base+path, bytes.NewReader(b))
	req.Header.Set("X-Orion-Token", a.token)
	req.Header.Set("Content-Type", "application/json")
	res, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", res.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}
	return nil
}

func healthy(s *session.Session) *api {
	a := &api{base: fmt.Sprintf("http://127.0.0.1:%d", s.Port), token: s.Token, http: &http.Client{Timeout: 2 * time.Second}}
	req, _ := http.NewRequest("GET", a.base+"/api/health", nil)
	req.Header.Set("X-Orion-Token", a.token)
	res, err := a.http.Do(req)
	if err != nil {
		return nil
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		return nil
	}
	return a
}

// Existing devolve a sessão do painel que já está rodando, se houver um vivo.
func Existing() (*session.Session, bool) {
	s, err := session.Load()
	if err != nil || healthy(s) == nil {
		return nil, false
	}
	return s, true
}

// connect acha o servidor pelo session.json; se não houver um vivo, sobe um em segundo plano.
func connect() *api {
	if s, err := session.Load(); err == nil {
		if a := healthy(s); a != nil {
			return a
		}
	}
	if err := startServer(); err != nil {
		fmt.Fprintln(os.Stderr, "orion: não consegui iniciar o painel:", err)
		return nil
	}
	deadline := time.Now().Add(waitServer)
	for time.Now().Before(deadline) {
		if s, err := session.Load(); err == nil {
			if a := healthy(s); a != nil {
				fmt.Fprintf(os.Stderr, "orion: painel em http://127.0.0.1:%d/?t=%s\n", s.Port, s.Token)
				return a
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

type line struct {
	Stream string `json:"stream"`
	Text   string `json:"text"`
}

// sender junta linhas e envia em lote (a cada 200 ms ou 200 linhas). Se o servidor cair,
// para de enviar em silêncio: o comando continua no terminal.
type sender struct {
	a       *api
	id      string
	mu      sync.Mutex
	buf     []line
	dead    bool
	kick    chan struct{}
	done    chan struct{}
	stopped chan struct{}
}

func newSender(a *api) *sender {
	s := &sender{a: a, kick: make(chan struct{}, 1), done: make(chan struct{}), stopped: make(chan struct{})}
	go s.loop()
	return s
}

// setID libera o envio; id vazio significa que o registro falhou e nada será enviado.
func (s *sender) setID(id string) {
	s.mu.Lock()
	s.id, s.dead = id, id == ""
	if s.dead {
		s.buf = nil
	}
	s.mu.Unlock()
}

func (s *sender) add(stream, text string) {
	s.mu.Lock()
	if !s.dead {
		s.buf = append(s.buf, line{stream, text})
	}
	full := len(s.buf) >= 200
	s.mu.Unlock()
	if full {
		select {
		case s.kick <- struct{}{}:
		default:
		}
	}
}

func (s *sender) flush() {
	s.mu.Lock()
	if s.id == "" {
		s.mu.Unlock()
		return
	}
	batch, id := s.buf, s.id
	s.buf = nil
	s.mu.Unlock()
	if len(batch) == 0 {
		return
	}
	if s.a.post("/api/runs/"+id+"/lines", batch, nil) != nil {
		s.mu.Lock()
		s.dead = true
		s.mu.Unlock()
	}
}

func (s *sender) loop() {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
		case <-s.kick:
		case <-s.done:
			s.flush()
			close(s.stopped)
			return
		}
		s.flush()
	}
}

// close envia o que sobrou e espera o loop terminar.
func (s *sender) close() {
	close(s.done)
	<-s.stopped
}

// Run roda o comando e devolve o código de saída dele.
func Run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "uso: orion run <comando> [argumentos...]")
		return 2
	}
	a := connect()
	if a == nil {
		fmt.Fprintln(os.Stderr, "orion: painel indisponível; rodando sem enviar logs")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	var s *sender
	var outW, errW io.WriteCloser
	if a != nil {
		s = newSender(a) // guarda as linhas até saber o id do run
		outW, errW = logs.LineWriter("out", s.add), logs.LineWriter("err", s.add)
		cmd.Stdout, cmd.Stderr = io.MultiWriter(os.Stdout, outW), io.MultiWriter(os.Stderr, errW)
	}
	if err := cmd.Start(); err != nil {
		if s != nil {
			s.close()
		}
		return startFailed(err)
	}
	if s != nil {
		var res struct{ ID string }
		if err := a.post("/api/runs", map[string]any{"cmd": strings.Join(args, " "), "pid": cmd.Process.Pid}, &res); err != nil {
			s.setID("") // servidor sumiu: descarta
		} else {
			s.setID(res.ID)
		}
	}

	// Ctrl+C chega ao filho pelo console; aqui só não deixamos o orion morrer antes dele.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	err := cmd.Wait()
	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	} else if err != nil {
		code = 1
	}
	if s != nil {
		outW.Close()
		errW.Close()
		s.close()
		if !s.dead {
			s.a.post("/api/runs/"+s.id+"/exit", map[string]int{"code": code}, nil)
		}
	}
	return code
}

func startFailed(err error) int {
	fmt.Fprintln(os.Stderr, "orion:", err)
	if errors.Is(err, exec.ErrNotFound) {
		return 9009 // mesmo código que o cmd usa para "comando não encontrado"
	}
	return 1
}
