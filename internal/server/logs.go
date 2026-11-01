package server

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/SamuelSchutz13/orion-logger/internal/logs"
	"github.com/gin-gonic/gin"
)

var lineSeq atomic.Int64

// GET /api/logs?id=<item id> — SSE com eventos "lines" (lote de logs.Line).
func streamLogs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Query("id")
		ctx := c.Request.Context()
		ch := make(chan logs.Line, 4096)
		emit := func(stream, text string) {
			l := logs.Line{Seq: lineSeq.Add(1), Time: time.Now(), Stream: stream, Text: logs.Clean(text)}
			select {
			case ch <- l:
			case <-ctx.Done():
			}
		}
		// run alimenta ch até o ctx acabar; cada tipo de item tem a sua origem de log.
		var run func(ctx context.Context)
		var backlog []logs.Line
		switch kind, rest, _ := strings.Cut(id, ":"); {
		case kind == "run" && d.Hub != nil:
			back, sub, cancel := d.Hub.Subscribe(id)
			defer cancel()
			backlog = back
			run = func(ctx context.Context) {
				for {
					select {
					case l := <-sub:
						select {
						case ch <- l:
						case <-ctx.Done():
							return
						}
					case <-ctx.Done():
						return
					}
				}
			}
		case kind == "docker" && d.Docker != nil:
			run = until(func(ctx context.Context) error { return d.Docker.StreamLogs(ctx, rest, emit) }, emit)
		case kind == "k8s" && strings.HasPrefix(rest, "pod:") && d.Kube != nil:
			ns, name, _ := strings.Cut(strings.TrimPrefix(rest, "pod:"), "/")
			run = until(func(ctx context.Context) error { return d.Kube.StreamLogs(ctx, ns, name, emit) }, emit)
		case kind == "file" && d.LogFiles != nil && d.LogFiles.Has(rest):
			run = func(ctx context.Context) { logs.Tail(ctx, rest, emit) }
		default:
			c.JSON(404, gin.H{"error": "este item não tem log"})
			return
		}
		go run(ctx)

		send := sse(c)
		if len(backlog) > 0 && send("lines", backlog) != nil {
			return
		}
		keep := time.NewTicker(15 * time.Second)
		defer keep.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-keep.C:
				if ping(c) != nil {
					return
				}
			case l := <-ch:
				// junta o que já chegou num lote só: menos eventos com logs muito rápidos
				batch := []logs.Line{l}
				for len(batch) < 500 {
					select {
					case l := <-ch:
						batch = append(batch, l)
						continue
					default:
					}
					break
				}
				if send("lines", batch) != nil {
					return
				}
			}
		}
	}
}

// until roda um stream que pode terminar (container parou, pod sumiu) e avisa no log.
// A conexão SSE continua aberta, para o navegador não reconectar e repetir as últimas linhas.
func until(f func(context.Context) error, emit logs.Emit) func(context.Context) {
	return func(ctx context.Context) {
		err := f(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			emit("sys", "log indisponível: "+err.Error())
		} else {
			emit("sys", "stream de log encerrado")
		}
	}
}
