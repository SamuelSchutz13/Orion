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
