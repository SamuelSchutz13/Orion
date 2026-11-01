// Package actions encerra processos com segurança: confere o PID antes de agir.
package actions

import (
	"errors"
	"fmt"
	"os"

	"github.com/shirou/gopsutil/v4/process"
)

var (
	ErrChanged = errors.New("o processo mudou ou já terminou, atualize a lista")
	ErrDenied  = errors.New("sem permissão para encerrar (processo de sistema ou de outro usuário)")
)

type PreviewInfo struct {
	Name     string `json:"name"`
	PID      int32  `json:"pid"`
	Children int    `json:"children"`
}

// open só devolve o processo se o PID ainda for o mesmo processo listado (mesmo horário de criação).
func open(pid int32, createMs int64) (*process.Process, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, ErrChanged
	}
	ct, err := p.CreateTime()
	if err != nil {
		if ok, _ := process.PidExists(pid); ok {
			// Existe mas não dá para ler (processo protegido): sem confirmar a identidade, não encerra.
			return nil, ErrDenied
		}
		return nil, ErrChanged
	}
	if ct != createMs {
		return nil, ErrChanged
	}
	return p, nil
}

// tree devolve todos os descendentes. Filhos criados antes do pai são de um PID reutilizado e ficam de fora.
func tree(p *process.Process) []*process.Process {
	parentCT, _ := p.CreateTime()
	kids, _ := p.Children()
	var out []*process.Process
	for _, k := range kids {
		if ct, err := k.CreateTime(); err != nil || ct < parentCT {
			continue
		}
		out = append(out, k)
		out = append(out, tree(k)...)
	}
	return out
}

func Preview(pid int32, createMs int64) (PreviewInfo, error) {
	p, err := open(pid, createMs)
	if err != nil {
		return PreviewInfo{}, err
	}
	name, _ := p.Name()
	return PreviewInfo{Name: name, PID: pid, Children: len(tree(p))}, nil
}

// KillTree encerra o processo e toda a árvore abaixo dele (como taskkill /T /F).
// O pai morre primeiro para não recriar os filhos.
func KillTree(pid int32, createMs int64) error {
	p, err := open(pid, createMs)
	if err != nil {
		return err
	}
	kids := tree(p)
	if err := p.Kill(); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return ErrDenied
		}
		if ok, _ := process.PidExists(pid); !ok {
			return ErrChanged
		}
		return fmt.Errorf("não foi possível encerrar: %w", err)
	}
	for _, k := range kids {
		k.Kill() // filho que já saiu sozinho não é erro
	}
	return nil
}
