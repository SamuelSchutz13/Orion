import type { Item } from './api'

export interface Hist { cpu: number[]; mem: number[] }

const N = 30 // 60 s com atualização a cada 2 s

// Guarda as últimas amostras de CPU (%) e memória (MB) de cada processo, só na memória do navegador.
// Itens que sumiram da lista saem do mapa.
export function updateHistory(prev: Map<string, Hist>, items: Item[]): Map<string, Hist> {
  const next = new Map<string, Hist>()
  for (const i of items) {
    const cpu = parseFloat(i.detail?.cpu ?? ''), mem = parseFloat(i.detail?.mem ?? '')
    if (Number.isNaN(cpu) || Number.isNaN(mem)) continue
    const h = prev.get(i.id) ?? { cpu: [], mem: [] }
    next.set(i.id, { cpu: [...h.cpu, cpu].slice(-N), mem: [...h.mem, mem].slice(-N) })
  }
  return next
}
