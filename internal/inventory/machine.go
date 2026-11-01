package inventory

import (
	"context"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// MachineStats lê CPU (desde a leitura anterior) e memória da máquina; nil se falhar.
func MachineStats(ctx context.Context) *Machine {
	m := &Machine{}
	if p, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(p) == 1 {
		m.CPU = p[0]
	}
	m.Cores, _ = cpu.CountsWithContext(ctx, true)
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil
	}
	m.MemUsed, m.MemTotal = v.Used, v.Total
	return m
}
