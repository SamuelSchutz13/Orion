package logs

import (
	"bytes"
	"strings"
	"sync"
)

// Emit recebe uma linha já separada: stream é "out", "err" ou "sys".
type Emit func(stream, text string)

type lineWriter struct {
	mu     sync.Mutex
	stream string
	emit   Emit
	buf    []byte
}

// LineWriter corta o que recebe em linhas (sem \r\n) e chama emit para cada uma.
// Close entrega a última linha incompleta.
func LineWriter(stream string, emit Emit) interface {
	Write([]byte) (int, error)
	Close() error
} {
	return &lineWriter{stream: stream, emit: emit}
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		w.emit(w.stream, strings.TrimRight(string(w.buf[:i]), "\r"))
		w.buf = w.buf[i+1:]
	}
	// shortcut: linha sem \n acima de 64 KB é cortada, para não crescer sem limite.
	if len(w.buf) > 64*1024 {
		w.emit(w.stream, string(w.buf))
		w.buf = nil
	}
	return len(p), nil
}

func (w *lineWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) > 0 {
		w.emit(w.stream, strings.TrimRight(string(w.buf), "\r"))
		w.buf = nil
	}
	return nil
}
