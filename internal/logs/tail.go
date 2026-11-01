package logs

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"
)

const (
	tailLines    = 500
	tailMaxStart = 4 << 20 // ao começar, lê no máximo os últimos 4 MB
)

// Tail acompanha um arquivo como `tail -f`: começa nas últimas 500 linhas, recomeça do início
// se o arquivo for truncado/rotacionado e espera o arquivo aparecer se ele não existir.
// O arquivo é reaberto a cada leitura para não travar rotação/exclusão no Windows.
// shortcut: rotação para um arquivo novo já maior que o offset não é detectada; comparar o file ID se virar problema.
func Tail(ctx context.Context, path string, emit Emit) {
	var (
		offset  int64 = -1 // -1: ainda não começou
		missing bool
		w       = LineWriter("out", emit)
	)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		fi, err := os.Stat(path)
		switch {
		case err != nil:
			if !missing {
				emit("sys", "arquivo não encontrado: "+path)
				missing = true
			}
			offset = -1
		case offset < 0:
			missing = false
			offset = readStart(path, fi.Size(), emit)
		default:
			if fi.Size() < offset {
				emit("sys", "arquivo truncado, recomeçando do início")
				w = LineWriter("out", emit)
				offset = 0
			}
			if fi.Size() > offset {
				offset = readFrom(path, offset, w)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// readStart emite as últimas tailLines linhas completas e devolve onde continuar.
func readStart(path string, size int64, emit Emit) int64 {
	f, err := os.Open(path)
	if err != nil {
		return -1
	}
	defer f.Close()
	from := max(size-tailMaxStart, 0)
	b := make([]byte, size-from)
	n, _ := f.ReadAt(b, from)
	b = b[:n]
	end := bytes.LastIndexByte(b, '\n') + 1 // linha incompleta no fim fica para a próxima leitura
	lines := bytes.Split(b[:end], []byte{'\n'})
	lines = lines[:len(lines)-1]
	if from > 0 && len(lines) > 0 {
		lines = lines[1:] // a primeira provavelmente começou no meio
	}
	if len(lines) > tailLines {
		lines = lines[len(lines)-tailLines:]
	}
	for _, l := range lines {
		emit("out", string(bytes.TrimRight(l, "\r")))
	}
	return from + int64(end)
}

func readFrom(path string, offset int64, w io.Writer) int64 {
	f, err := os.Open(path)
	if err != nil {
		return offset
	}
	defer f.Close()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return offset
	}
	n, _ := io.Copy(w, f)
	return offset + n
}
