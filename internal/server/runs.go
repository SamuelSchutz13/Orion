package server

import (
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/SamuelSchutz13/orion-logger/internal/inventory"
	"github.com/SamuelSchutz13/orion-logger/internal/logs"
	"github.com/shirou/gopsutil/v4/process"
)

type run struct {
	id       string
	cmd      string
	pid      int32
	createMs int64
	started  time.Time
	exited   bool
	code     int
}

// Runs guarda, só na memória da sessão, os comandos iniciados com `orion run`.
type Runs struct {
	mu  sync.Mutex
	n   int
	m   map[string]*run
	hub *logs.Hub
}

func NewRuns(hub *logs.Hub) *Runs { return &Runs{m: map[string]*run{}, hub: hub} }

func (r *Runs) Start(cmd string, pid int32) string {
	var ct int64
	if p, err := process.NewProcess(pid); err == nil {
		ct, _ = p.CreateTime()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.n++
	id := fmt.Sprintf("run:%d", r.n)
	r.m[id] = &run{id: id, cmd: cmd, pid: pid, createMs: ct, started: time.Now()}
	return id
}

func (r *Runs) Exit(id string, code int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	x, ok := r.m[id]
	if ok {
		x.exited, x.code = true, code
	}
	return ok
}

func (r *Runs) Has(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.m[id]
	return ok
}

func (r *Runs) proc(id string) (pid int32, createMs int64, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	x, ok := r.m[id]
	if !ok || x.exited {
		return 0, 0, false
	}
	return x.pid, x.createMs, true
}

func (r *Runs) Dismiss(id string) {
	r.mu.Lock()
	delete(r.m, id)
	r.mu.Unlock()
	r.hub.Drop(id)
}

// Merge acrescenta os itens run: e absorve a linha proc: do mesmo processo (portas, CPU, memória),
// para cada processo aparecer uma vez só.
// shortcut: só o PID iniciado é absorvido; netos (npm → node) continuam em linhas próprias.
func (r *Runs) Merge(items []inventory.Item) []inventory.Item {
	r.mu.Lock()
	runs := make([]run, 0, len(r.m))
	for _, x := range r.m {
		runs = append(runs, *x)
	}
	r.mu.Unlock()
	slices.SortFunc(runs, func(a, b run) int { return a.started.Compare(b.started) })
	for _, x := range runs {
		it := inventory.Item{
			ID: x.id, Kind: inventory.KindRun, Name: x.cmd, Status: inventory.StatusOK, Ports: []int{},
			PID: int(x.pid), HasLogs: true, Actions: []string{"kill"},
			Detail: map[string]string{"cmdline": x.cmd, "started": x.started.Format("15:04:05")},
		}
		if x.exited {
			it.Status, it.Actions = inventory.StatusDown, []string{"dismiss"}
			it.Detail["exitCode"] = strconv.Itoa(x.code)
		} else {
			procID := fmt.Sprintf("proc:%d:%d", x.pid, x.createMs)
			if i := slices.IndexFunc(items, func(p inventory.Item) bool { return p.ID == procID }); i >= 0 {
				it.Ports = items[i].Ports
				for _, k := range []string{"path", "cpu", "mem", "user"} {
					it.Detail[k] = items[i].Detail[k]
				}
				items = slices.Delete(items, i, i+1)
			}
		}
		items = append(items, it)
	}
	return items
}
