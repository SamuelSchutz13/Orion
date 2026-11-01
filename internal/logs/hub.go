// Package logs guarda e acompanha linhas de log (orion run, arquivos .log, containers, pods).
package logs

import (
	"fmt"
	"sync"
	"time"
)

type Line struct {
	Seq    int64     `json:"seq"`
	Time   time.Time `json:"time"`
	Stream string    `json:"stream"` // "out", "err" ou "sys"
	Text   string    `json:"text"`
}

type sub struct {
	ch      chan Line
	skipped int
}

type stream struct {
	lines []Line // buffer circular
	start int
	subs  map[*sub]struct{}
}

// Hub mantém as últimas N linhas de cada stream e repassa as novas aos assinantes.
type Hub struct {
	mu      sync.Mutex
	cap     int
	seq     int64
	streams map[string]*stream
}

func NewHub(capPerStream int) *Hub { return &Hub{cap: capPerStream, streams: map[string]*stream{}} }

func (h *Hub) get(id string) *stream {
	s := h.streams[id]
	if s == nil {
		s = &stream{subs: map[*sub]struct{}{}}
		h.streams[id] = s
	}
	return s
}

func (h *Hub) Append(id, streamName, text string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.seq++
	l := Line{Seq: h.seq, Time: time.Now(), Stream: streamName, Text: Clean(text)}
	s := h.get(id)
	if len(s.lines) < h.cap {
		s.lines = append(s.lines, l)
	} else {
		s.lines[s.start] = l
		s.start = (s.start + 1) % h.cap
	}
	for su := range s.subs {
		if su.skipped > 0 && len(su.ch) < cap(su.ch)-1 {
			su.ch <- Line{Seq: l.Seq, Time: l.Time, Stream: "sys", Text: fmt.Sprintf("%d linhas puladas (navegador lento)", su.skipped)}
			su.skipped = 0
		}
		select {
		case su.ch <- l:
		default:
			su.skipped++ // nunca bloqueia quem escreve
		}
	}
}

func (h *Hub) Subscribe(id string) (backlog []Line, ch <-chan Line, cancel func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.get(id)
	backlog = append(append([]Line{}, s.lines[s.start:]...), s.lines[:s.start]...)
	su := &sub{ch: make(chan Line, 1024)}
	s.subs[su] = struct{}{}
	return backlog, su.ch, func() {
		h.mu.Lock()
		delete(s.subs, su)
		h.mu.Unlock()
	}
}

// Drop apaga o stream (ao dispensar um item do orion run).
func (h *Hub) Drop(id string) {
	h.mu.Lock()
	delete(h.streams, id)
	h.mu.Unlock()
}
