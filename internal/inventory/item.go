// Package inventory lista o que está rodando: processos, portas, Docker e Kubernetes.
package inventory

import "context"

type Kind string

const (
	KindProc    Kind = "proc"
	KindDocker  Kind = "docker"
	KindPod     Kind = "pod"
	KindService Kind = "service"
	KindDeploy  Kind = "deployment"
	KindLogFile Kind = "logfile"
	KindRun     Kind = "run"
)

type Status string

const (
	StatusOK   Status = "ok"
	StatusWarn Status = "warn"
	StatusDown Status = "down"
)

type Item struct {
	ID        string            `json:"id"`
	Kind      Kind              `json:"kind"`
	Name      string            `json:"name"`
	Status    Status            `json:"status"`
	Ports     []int             `json:"ports"`
	PID       int               `json:"pid,omitempty"`
	Namespace string            `json:"namespace,omitempty"`
	Image     string            `json:"image,omitempty"`
	Detail    map[string]string `json:"detail,omitempty"`
	HasLogs   bool              `json:"hasLogs"`
	Actions   []string          `json:"actions"`
	Hidden    bool              `json:"hidden"`
}

type SourceState struct {
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
}

type Snapshot struct {
	Items   []Item                 `json:"items"`
	Sources map[string]SourceState `json:"sources"`
}

type Source interface {
	Name() string
	List(ctx context.Context) ([]Item, error)
}
