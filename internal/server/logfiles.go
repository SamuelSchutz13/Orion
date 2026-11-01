package server

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/SamuelSchutz13/orion-logger/internal/inventory"
	"github.com/SamuelSchutz13/orion-logger/internal/session"
)

// LogFiles é a lista de arquivos .log acompanhados, salva em config.json.
type LogFiles struct {
	mu  sync.Mutex
	cfg *session.Config
}

func NewLogFiles() *LogFiles { return &LogFiles{cfg: session.LoadConfig()} }

func (l *LogFiles) Paths() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.cfg.LogFiles)
}

func (l *LogFiles) Has(path string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.ContainsFunc(l.cfg.LogFiles, func(p string) bool { return strings.EqualFold(p, path) })
}

func (l *LogFiles) Add(path string) (string, error) {
	path = strings.Trim(strings.TrimSpace(path), `"`)
	if path == "" {
		return "", errors.New("informe o caminho do arquivo")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return "", errors.New("arquivo não encontrado: " + abs)
	}
	if fi.IsDir() {
		return "", errors.New("é uma pasta, não um arquivo: " + abs)
	}
	if l.Has(abs) {
		return abs, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cfg.LogFiles = append(l.cfg.LogFiles, abs)
	return abs, l.cfg.Save()
}

func (l *LogFiles) Remove(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cfg.LogFiles = slices.DeleteFunc(l.cfg.LogFiles, func(p string) bool { return strings.EqualFold(p, path) })
	return l.cfg.Save()
}

func (l *LogFiles) Items() []inventory.Item {
	var items []inventory.Item
	for _, p := range l.Paths() {
		st := inventory.StatusOK
		if _, err := os.Stat(p); err != nil {
			st = inventory.StatusDown
		}
		items = append(items, inventory.Item{
			ID: "file:" + p, Kind: inventory.KindLogFile, Name: filepath.Base(p), Status: st,
			Ports: []int{}, HasLogs: true, Actions: []string{"dismiss"}, Detail: map[string]string{"path": p},
		})
	}
	return items
}
