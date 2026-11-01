package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/SamuelSchutz13/orion-logger/internal/inventory"
	"github.com/shirou/gopsutil/v4/process"
)

type items []inventory.Item

func (items) Name() string                                     { return "fake" }
func (i items) List(context.Context) ([]inventory.Item, error) { return i, nil }

func collectorWith(t *testing.T, it ...inventory.Item) *inventory.Collector {
	c := inventory.NewCollector(time.Hour, nil, items(it))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go c.Run(ctx)
	for i := 0; i < 100; i++ {
		if len(c.Latest().Items) == len(it) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return c
}

func TestActionNotAllowed(t *testing.T) {
	c := collectorWith(t, inventory.Item{ID: "docker:x", Kind: inventory.KindDocker, Actions: []string{"stop"}})
	base, s := liveServer(t, Deps{Collector: c})
	if res := send(t, s, "POST", base+"/api/actions", map[string]string{"id": "docker:x", "action": "kill"}); res.StatusCode != 400 {
		t.Fatalf("%d", res.StatusCode)
	}
}

func TestUnknownID(t *testing.T) {
	base, s := liveServer(t, Deps{Collector: collectorWith(t)})
	if res := send(t, s, "POST", base+"/api/actions", map[string]string{"id": "proc:1:1", "action": "kill"}); res.StatusCode != 404 {
		t.Fatalf("%d", res.StatusCode)
	}
}

func TestKillViaAPI(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "ping -n 30 127.0.0.1 >NUL")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }()
	p, _ := process.NewProcess(int32(cmd.Process.Pid))
	ct, _ := p.CreateTime()
	id := fmt.Sprintf("proc:%d:%d", cmd.Process.Pid, ct)
	stale := fmt.Sprintf("proc:%d:%d", cmd.Process.Pid, ct-1000)
	c := collectorWith(t,
		inventory.Item{ID: id, Kind: inventory.KindProc, Actions: []string{"kill"}},
		inventory.Item{ID: stale, Kind: inventory.KindProc, Actions: []string{"kill"}})
	base, s := liveServer(t, Deps{Collector: c})

	res := get(t, s, base+"/api/actions/preview?id="+id)
	var info struct{ Name string }
	json.NewDecoder(res.Body).Decode(&info)
	if res.StatusCode != 200 || info.Name != "cmd.exe" {
		t.Fatalf("preview %d %+v", res.StatusCode, info)
	}
	if res := send(t, s, "POST", base+"/api/actions", map[string]string{"id": stale, "action": "kill"}); res.StatusCode != 409 {
		t.Fatalf("stale: %d", res.StatusCode)
	}
	if res := send(t, s, "POST", base+"/api/actions", map[string]string{"id": id, "action": "kill"}); res.StatusCode != 204 {
		t.Fatalf("kill: %d", res.StatusCode)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("processo não morreu")
	}
}

func TestRunsMergeAbsorbsProc(t *testing.T) {
	r := NewRuns(nil)
	r.m["run:1"] = &run{id: "run:1", cmd: "npm run dev", pid: 42, createMs: 7}
	out := r.Merge([]inventory.Item{{ID: "proc:42:7", Ports: []int{3000}, Detail: map[string]string{"cpu": "1.0%"}}, {ID: "proc:1:1"}})
	if len(out) != 2 || out[1].ID != "run:1" || out[1].Ports[0] != 3000 || out[1].Detail["cpu"] != "1.0%" {
		t.Fatalf("%+v", out)
	}
}

// npm (run) → cmd → node: a porta do node aparece na linha do run e os filhos somem da lista.
func TestRunsMergeAbsorbsDescendants(t *testing.T) {
	r := NewRuns(nil)
	r.m["run:1"] = &run{id: "run:1", cmd: "npm run dev", pid: 10, createMs: 100}
	d := func(cpu, mem string) map[string]string { return map[string]string{"cpu": cpu, "mem": mem} }
	in := []inventory.Item{
		{ID: "proc:10:100", PID: 10, PPID: 1, Ports: []int{}, Detail: d("0.5%", "10.0 MB")},
		{ID: "proc:11:110", PID: 11, PPID: 10, Ports: []int{}, Detail: d("0.0%", "2.0 MB")},
		{ID: "proc:12:120", PID: 12, PPID: 11, Ports: []int{5173, 24678}, Detail: d("3.0%", "80.5 MB")},
		{ID: "proc:13:50", PID: 13, PPID: 10, Ports: []int{9999}, Detail: d("1.0%", "1.0 MB")}, // criado antes do pai: PID reaproveitado, não é filho
		{ID: "proc:20:1", PID: 20, PPID: 1, Ports: []int{80}},
	}
	out := r.Merge(in)
	ids := []string{}
	for _, it := range out {
		ids = append(ids, it.ID)
	}
	if fmt.Sprint(ids) != "[proc:13:50 proc:20:1 run:1]" {
		t.Fatalf("%v", ids)
	}
	run := out[2]
	if fmt.Sprint(run.Ports) != "[5173 24678]" || run.Detail["cpu"] != "3.5%" || run.Detail["mem"] != "92.5 MB" || run.Detail["children"] != "2" {
		t.Fatalf("%+v", run)
	}
}
