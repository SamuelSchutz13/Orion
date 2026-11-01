package inventory

import (
	"context"
	"sync"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type procKey struct {
	pid      int32
	createMs int64
}

type cachedProc struct {
	p    *process.Process // reaproveitado para que Percent(0) meça contra a leitura anterior
	info ProcInfo         // campos que não mudam (nome, caminho, linha de comando, usuário)
}

type ProcSource struct {
	mu      sync.Mutex
	cache   map[procKey]*cachedProc
	runPIDs func() map[int32]bool
}

func NewProcSource() *ProcSource { return &ProcSource{cache: map[procKey]*cachedProc{}} }

func (s *ProcSource) Name() string { return "procs" }

// SetRunPIDs informa os PIDs iniciados pelo `orion run`, que ficam sempre visíveis.
func (s *ProcSource) SetRunPIDs(f func() map[int32]bool) { s.runPIDs = f }

func (s *ProcSource) List(ctx context.Context) ([]Item, error) {
	pids, err := process.PidsWithContext(ctx)
	if err != nil {
		return nil, err
	}
	names := snapshotNames()
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[procKey]bool{}
	infos := make([]ProcInfo, 0, len(pids))
	for _, pid := range pids {
		if pid == 0 {
			continue
		}
		p, err := process.NewProcessWithContext(ctx, pid)
		if err != nil {
			continue // saiu entre a listagem e agora
		}
		ct, _ := p.CreateTimeWithContext(ctx)
		k := procKey{pid, ct}
		seen[k] = true
		c := s.cache[k]
		if c == nil {
			// Sem permissão (processos do sistema) cada campo fica vazio; não falha a lista.
			info := ProcInfo{PID: pid, CreateMs: ct}
			if info.Name = names[pid]; info.Name == "" {
				info.Name, _ = p.NameWithContext(ctx)
			}
			info.Exe, _ = p.ExeWithContext(ctx)
			info.Cmdline, _ = p.CmdlineWithContext(ctx)
			info.Username, _ = p.UsernameWithContext(ctx)
			c = &cachedProc{p: p, info: info}
			s.cache[k] = c
		}
		info := c.info
		info.CPU, _ = c.p.PercentWithContext(ctx, 0)
		if m, err := c.p.MemoryInfoWithContext(ctx); err == nil {
			info.MemBytes = m.RSS
		}
		infos = append(infos, info)
	}
	for k := range s.cache {
		if !seen[k] {
			delete(s.cache, k)
		}
	}

	raw, err := gnet.ConnectionsWithContext(ctx, "inet")
	if err != nil {
		return nil, err
	}
	conns := make([]Conn, 0, len(raw))
	for _, c := range raw {
		tcp := c.Type == 1 // SOCK_STREAM
		if c.Pid == 0 || (tcp && c.Status != "LISTEN") {
			continue
		}
		proto := "udp"
		if tcp {
			proto = "tcp"
		}
		conns = append(conns, Conn{PID: c.Pid, Port: int(c.Laddr.Port), Proto: proto})
	}
	var run map[int32]bool
	if s.runPIDs != nil {
		run = s.runPIDs()
	}
	return AttachPorts(infos, conns, run), nil
}
