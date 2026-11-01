import { describe, expect, it } from 'vitest'
import type { Item } from './api'
import { updateHistory } from './history'

const it_ = (id: string, cpu?: string, mem?: string): Item => ({
  id, kind: 'proc', name: id, status: 'ok', ports: [], hasLogs: false, actions: [], hidden: false,
  detail: cpu ? { cpu, mem: mem! } : undefined,
})

describe('updateHistory', () => {
  it('acumula CPU e memória por item e limita a 30 amostras', () => {
    let h = new Map()
    for (let k = 0; k < 35; k++) h = updateHistory(h, [it_('a', `${k}.0%`, `${k * 10}.5 MB`)])
    const a = h.get('a')!
    expect(a.cpu.length).toBe(30)
    expect(a.cpu.at(-1)).toBe(34)
    expect(a.mem.at(-1)).toBe(340.5)
  })
  it('remove itens que sumiram e ignora itens sem CPU', () => {
    let h = updateHistory(new Map(), [it_('a', '1%', '1 MB'), it_('b', '2%', '2 MB'), it_('docker:x')])
    h = updateHistory(h, [it_('b', '3%', '3 MB')])
    expect([...h.keys()]).toEqual(['b'])
    expect(h.get('b')!.cpu).toEqual([2, 3])
  })
})
