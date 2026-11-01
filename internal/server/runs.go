package server

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
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

// Merge acrescenta os itens run: e absorve as linhas proc: do processo iniciado e de todos os
// descendentes (npm → cmd → node): portas juntas, CPU e memória somadas, uma linha só.
func (r *Runs) Merge(items []inventory.Item) []inventory.Item {
	r.mu.Lock()
	runs := make([]run, 0, len(r.m))
	for _, x := range r.m {
		runs = append(runs, *x)
	}
	r.mu.Unlock()
	slices.SortFunc(runs, func(a, b run) int { return a.started.Compare(b.started) })

	byID := map[string]int{}
	kids := map[int][]int{} // ppid → índices dos filhos
	for i, it := range items {
		byID[it.ID] = i
		if it.Kind == inventory.KindProc || it.Kind == "" {
			kids[it.PPID] = append(kids[it.PPID], i)
		}
	}
	absorbed := map[int]bool{}
	var out []inventory.Item
	for _, x := range runs {
		it := inventory.Item{
			ID: x.id, Kind: inventory.KindRun, Name: x.cmd, Status: inventory.StatusOK, Ports: []int{},
			PID: int(x.pid), HasLogs: true, Actions: []string{"kill"},
			Detail: map[string]string{"cmdline": x.cmd, "started": x.started.Format("15:04:05")},
		}
		if x.exited {
			it.Status, it.Actions = inventory.StatusDown, []string{"dismiss"}
			it.Detail["exitCode"] = strconv.Itoa(x.code)
		} else if root, ok := byID[fmt.Sprintf("proc:%d:%d", x.pid, x.createMs)]; ok {
			tree := descendants(items, kids, root)
			var cpu, mem float64
			for _, i := range tree {
				absorbed[i] = true
				for _, p := range items[i].Ports {
					if !slices.Contains(it.Ports, p) {
						it.Ports = append(it.Ports, p)
					}
				}
				cpu += num(items[i].Detail["cpu"])
				mem += num(items[i].Detail["mem"])
			}
			slices.Sort(it.Ports)
			for _, k := range []string{"path", "user"} {
				it.Detail[k] = items[root].Detail[k]
			}
			it.Detail["cpu"] = fmt.Sprintf("%.1f%%", cpu)
			it.Detail["mem"] = fmt.Sprintf("%.1f MB", mem)
			if len(tree) > 1 {
				it.Detail["children"] = strconv.Itoa(len(tree) - 1)
			}
		}
		out = append(out, it)
	}
	keep := make([]inventory.Item, 0, len(items)-len(absorbed)+len(out))
	for i, it := range items {
		if !absorbed[i] {
			keep = append(keep, it)
		}
	}
	return append(keep, out...)
}

// descendants devolve root e todos os descendentes. Um "filho" criado antes do pai é um PID
// reaproveitado (o pai de verdade já morreu) e fica de fora.
func descendants(items []inventory.Item, kids map[int][]int, root int) []int {
	tree := []int{root}
	for k := 0; k < len(tree); k++ {
		parent := items[tree[k]]
		for _, c := range kids[parent.PID] {
			if createMs(items[c].ID) >= createMs(parent.ID) && !slices.Contains(tree, c) {
				tree = append(tree, c)
			}
		}
	}
	return tree
}

func createMs(procID string) int64 {
	v, _ := strconv.ParseInt(procID[strings.LastIndexByte(procID, ':')+1:], 10, 64)
	return v
}

// num lê o número do início de "3.0%" ou "80.5 MB".
func num(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(strings.TrimRight(s, "%MB ")), 64)
	return v
}
