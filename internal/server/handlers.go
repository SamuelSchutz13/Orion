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
	}
	api.GET("/logs", streamLogs(d))
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
