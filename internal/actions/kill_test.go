package actions

import (
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

func startTree(t *testing.T) (int32, int64) {
	t.Helper()
	cmd := exec.Command("cmd", "/c", "ping -n 30 127.0.0.1 >NUL")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })
	go cmd.Wait()
	pid := int32(cmd.Process.Pid)
	p, _ := process.NewProcess(pid)
	ct, _ := p.CreateTime()
	// espera o ping (filho) aparecer
	for i := 0; i < 50; i++ {
		if ch, _ := p.Children(); len(ch) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	return pid, ct
}

func TestKillTreeKillsChildren(t *testing.T) {
	pid, ct := startTree(t)
	info, err := Preview(pid, ct)
	if err != nil || info.Children < 1 || info.Name != "cmd.exe" || info.PID != pid {
		t.Fatalf("%+v %v", info, err)
	}
	p, _ := process.NewProcess(pid)
	kids, _ := p.Children()
	if err := KillTree(pid, ct); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		alive := false
		for _, q := range append(kids, p) {
			if ok, _ := process.PidExists(q.Pid); ok {
				if c, _ := q.CreateTime(); c != 0 {
					alive = true
				}
			}
		}
		if !alive {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("pai ou filho ainda vivo")
}

func TestKillChangedPID(t *testing.T) {
	pid, ct := startTree(t)
	if err := KillTree(pid, ct+1); !errors.Is(err, ErrChanged) {
		t.Fatalf("err %v", err)
	}
	if _, err := Preview(pid, ct+1); !errors.Is(err, ErrChanged) {
		t.Fatalf("preview err %v", err)
	}
	if ok, _ := process.PidExists(pid); !ok {
		t.Fatal("processo não devia ter morrido")
	}
}

func TestKillMissingPID(t *testing.T) {
	if err := KillTree(99999999, 1); !errors.Is(err, ErrChanged) {
		t.Fatalf("err %v", err)
	}
}

// O Windows nunca deixa terminar o processo System (PID 4), nem como administrador.
func TestKillSystemDenied(t *testing.T) {
	p, err := process.NewProcess(4)
	if err != nil {
		t.Skip("sem PID 4")
	}
	ct, _ := p.CreateTime()
	if err := KillTree(4, ct); !errors.Is(err, ErrDenied) {
		t.Fatalf("err %v", err)
	}
}
