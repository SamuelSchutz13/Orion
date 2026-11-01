package inventory

import (
	"reflect"
	"testing"
)

func one(t *testing.T, items []Item) Item {
	t.Helper()
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	return items[0]
}

func TestPortsGroupedPerProcess(t *testing.T) {
	procs := []ProcInfo{{PID: 10, CreateMs: 555, Name: "node.exe"}}
	conns := []Conn{{PID: 10, Port: 3001, Proto: "tcp"}, {PID: 10, Port: 3000, Proto: "tcp"}, {PID: 10, Port: 3000, Proto: "udp"}}
	it := one(t, AttachPorts(procs, conns, nil))
	if it.ID != "proc:10:555" || !reflect.DeepEqual(it.Ports, []int{3000, 3001}) {
		t.Fatalf("%+v", it)
	}
}

func TestProcWithPortVisible(t *testing.T) {
	it := one(t, AttachPorts([]ProcInfo{{PID: 5, Name: "svchost.exe"}}, []Conn{{PID: 5, Port: 135, Proto: "tcp"}}, nil))
	if it.Hidden || !reflect.DeepEqual(it.Actions, []string{"kill"}) || it.HasLogs || it.Status != StatusOK || it.Kind != KindProc || it.PID != 5 {
		t.Fatalf("%+v", it)
	}
}

func TestDevRuntimeVisibleWithoutPort(t *testing.T) {
	if one(t, AttachPorts([]ProcInfo{{PID: 1, Name: "node.exe"}}, nil, nil)).Hidden {
		t.Fatal("node should be visible")
	}
	if !one(t, AttachPorts([]ProcInfo{{PID: 2, Name: "svchost.exe"}}, nil, nil)).Hidden {
		t.Fatal("svchost should be hidden")
	}
	if one(t, AttachPorts([]ProcInfo{{PID: 2, Name: "svchost.exe"}}, nil, map[int32]bool{2: true})).Hidden {
		t.Fatal("run pid should be visible")
	}
}

func TestIsDevRuntime(t *testing.T) {
	for _, n := range []string{"node.exe", "go.exe", "python.exe", "python3.exe", "dotnet.exe", "java.exe", "deno.exe", "bun.exe", "ruby.exe", "php.exe", "npm.cmd", "NODE.EXE"} {
		if !IsDevRuntime(n) {
			t.Errorf("%s should be dev runtime", n)
		}
	}
	if IsDevRuntime("explorer.exe") {
		t.Error("explorer is not a dev runtime")
	}
}

func TestDockerRelayLosesPorts(t *testing.T) {
	it := one(t, AttachPorts([]ProcInfo{{PID: 9, Name: "com.docker.backend.exe"}}, []Conn{{PID: 9, Port: 8080, Proto: "tcp"}}, nil))
	if len(it.Ports) != 0 || !it.Hidden {
		t.Fatalf("%+v", it)
	}
}
