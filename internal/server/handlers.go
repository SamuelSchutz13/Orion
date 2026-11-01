package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func registerAPI(api *gin.RouterGroup, d Deps) {
	if d.Collector != nil {
		api.GET("/stream", streamSnapshots(d))
		api.GET("/actions/preview", previewAction(d))
		api.POST("/actions", doAction(d))
	}
	api.GET("/logs", streamLogs(d))
	if d.Runs != nil && d.Hub != nil {
		api.POST("/runs", func(c *gin.Context) {
			var body struct {
				Cmd string
				PID int32
			}
			if c.ShouldBindJSON(&body) != nil || body.PID <= 0 {
				c.JSON(400, gin.H{"error": "JSON inválido"})
				return
			}
			id := d.Runs.Start(body.Cmd, body.PID)
			refresh(d)
			c.JSON(200, gin.H{"id": id})
		})
		api.POST("/runs/:id/lines", func(c *gin.Context) {
			var lines []struct{ Stream, Text string }
			if c.ShouldBindJSON(&lines) != nil {
				c.JSON(400, gin.H{"error": "JSON inválido"})
				return
			}
			id := c.Param("id")
			if !d.Runs.Has(id) {
				c.JSON(404, gin.H{"error": "run desconhecido"})
				return
			}
			for _, l := range lines {
				if l.Stream != "err" {
					l.Stream = "out"
				}
				d.Hub.Append(id, l.Stream, l.Text)
			}
			c.Status(204)
		})
		api.POST("/runs/:id/exit", func(c *gin.Context) {
			var body struct{ Code int }
			if c.ShouldBindJSON(&body) != nil {
				c.JSON(400, gin.H{"error": "JSON inválido"})
				return
			}
			id := c.Param("id")
			if !d.Runs.Exit(id, body.Code) {
				c.JSON(404, gin.H{"error": "run desconhecido"})
				return
			}
			d.Hub.Append(id, "sys", fmt.Sprintf("processo terminou com código %d", body.Code))
			refresh(d)
			c.Status(204)
		})
	}
	if d.LogFiles != nil {
		api.GET("/logfiles", func(c *gin.Context) { c.JSON(200, d.LogFiles.Paths()) })
		api.POST("/logfiles", func(c *gin.Context) {
			var body struct{ Path string }
			if c.ShouldBindJSON(&body) != nil {
				c.JSON(400, gin.H{"error": "JSON inválido"})
				return
			}
			p, err := d.LogFiles.Add(body.Path)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			refresh(d)
			c.JSON(200, gin.H{"path": p})
		})
	}
}

func refresh(d Deps) {
	if d.Collector != nil {
		d.Collector.Refresh()
	}
}

// sse prepara a resposta como Server-Sent Events e devolve uma função que envia um evento.
func sse(c *gin.Context) func(event string, v any) error {
	h := c.Writer.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	c.Status(200)
	c.Writer.Flush()
	return func(event string, v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, b); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}
}

func ping(c *gin.Context) error {
	if _, err := fmt.Fprint(c.Writer, ": ping\n\n"); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func streamSnapshots(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ch, unsub := d.Collector.Subscribe()
		defer unsub()
		send := sse(c)
		keep := time.NewTicker(15 * time.Second)
		defer keep.Stop()
		for {
			select {
			case <-c.Request.Context().Done():
				return
			case s := <-ch:
				if send("snapshot", s) != nil {
					return
				}
			case <-keep.C:
				if ping(c) != nil {
					return
				}
			}
		}
	}
}
