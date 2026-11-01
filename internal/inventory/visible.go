package inventory

import (
	"fmt"
	"slices"
	"strings"
)

type ProcInfo struct {
	PID      int32
	PPID     int32
	CreateMs int64
	Name     string
	Exe      string
	Cmdline  string
	CPU      float64
	MemBytes uint64
	Username string
}

type Conn struct {
	PID   int32
	Port  int
	Proto string
}

var devRuntimes = map[string]bool{
	"node": true, "go": true, "python": true, "python3": true, "pythonw": true, "dotnet": true,
	"java": true, "javaw": true, "deno": true, "bun": true, "ruby": true, "php": true,
	"npm": true, "npx": true, "pnpm": true, "yarn": true, "cargo": true, "air": true,
	"uvicorn": true, "gunicorn": true, "rails": true, "nginx": true, "caddy": true,
}

// Esses processos só repassam as portas dos containers; a porta aparece na linha do container.
var dockerRelays = map[string]bool{
	"com.docker.backend.exe": true, "wslrelay.exe": true, "vpnkit.exe": true, "com.docker.proxy.exe": true,
}

func IsDevRuntime(name string) bool {
	n := strings.ToLower(name)
	for _, ext := range []string{".exe", ".cmd", ".bat"} {
		n = strings.TrimSuffix(n, ext)
	}
	return devRuntimes[n]
}

func AttachPorts(procs []ProcInfo, conns []Conn, runPIDs map[int32]bool) []Item {
	ports := map[int32][]int{}
	for _, c := range conns {
		if !slices.Contains(ports[c.PID], c.Port) {
			ports[c.PID] = append(ports[c.PID], c.Port)
		}
	}
	items := make([]Item, 0, len(procs))
	for _, p := range procs {
		relay := dockerRelays[strings.ToLower(p.Name)]
		var ps []int
		if !relay {
			ps = ports[p.PID]
			slices.Sort(ps)
		}
		if ps == nil {
			ps = []int{}
		}
		items = append(items, Item{
			ID:      fmt.Sprintf("proc:%d:%d", p.PID, p.CreateMs),
			Kind:    KindProc,
			Name:    p.Name,
			Status:  StatusOK,
			Ports:   ps,
			PID:     int(p.PID),
			PPID:    int(p.PPID),
			Actions: []string{"kill"},
			Hidden:  relay || !(len(ps) > 0 || runPIDs[p.PID] || IsDevRuntime(p.Name)),
			Detail: map[string]string{
				"path":    p.Exe,
				"cmdline": p.Cmdline,
				"cpu":     fmt.Sprintf("%.1f%%", p.CPU),
				"mem":     fmt.Sprintf("%.1f MB", float64(p.MemBytes)/1024/1024),
				"user":    p.Username,
			},
		})
	}
	return items
}
