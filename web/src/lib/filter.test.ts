import { describe, expect, it } from 'vitest'
import type { Item } from './api'
import { chipCount, filterItems, isSystem } from './filter'

const base = { status: 'ok', hasLogs: false, actions: [], hidden: false, ports: [] } as const
const items: Item[] = [
  { ...base, id: 'proc:1234:1', kind: 'proc', name: 'node.exe', pid: 1234, ports: [3000] },
  { ...base, id: 'docker:a', kind: 'docker', name: 'cache', image: 'redis:7', hasLogs: true },
  { ...base, id: 'k8s:pod:kube-system/coredns', kind: 'pod', name: 'coredns', namespace: 'kube-system', hidden: true },
  { ...base, id: 'k8s:svc:default/web', kind: 'service', name: 'web', namespace: 'default', ports: [80] },
  { ...base, id: 'k8s:deploy:default/web', kind: 'deployment', name: 'web', namespace: 'default' },
  { ...base, id: 'proc:4:0', kind: 'proc', name: 'svchost.exe', pid: 4, hidden: true },
  { ...base, id: 'file:C:\\a.log', kind: 'logfile', name: 'a.log', hasLogs: true },
  { ...base, id: 'run:1', kind: 'run', name: 'npm run dev', hasLogs: true },
]
const ids = (r: Item[]) => r.map(i => i.id)
const f = (q: string, chip: Parameters<typeof filterItems>[2] = 'all', showAll = true, showSystem = true) =>
  ids(filterItems(items, q, chip, showAll, showSystem))

describe('filterItems', () => {
  it('busca por porta, namespace, PID e imagem, sem diferenciar maiúsculas', () => {
    expect(f('3000')).toEqual(['proc:1234:1'])
    expect(f('KUBE')).toEqual(['k8s:pod:kube-system/coredns'])
    expect(f('1234')).toEqual(['proc:1234:1'])
    expect(f('Redis')).toEqual(['docker:a'])
  })

  it('chips por tipo', () => {
    expect(f('', 'ports')).toEqual(['proc:1234:1', 'k8s:svc:default/web'])
    expect(f('', 'procs')).toEqual(['proc:1234:1', 'proc:4:0', 'run:1'])
    expect(f('', 'docker')).toEqual(['docker:a'])
    expect(f('', 'k8s')).toEqual(['k8s:pod:kube-system/coredns', 'k8s:svc:default/web', 'k8s:deploy:default/web'])
    expect(f('', 'logs')).toEqual(['file:C:\\a.log', 'run:1'])
  })

  it('itens ocultos dependem dos dois botões', () => {
    expect(f('', 'all', false, false)).not.toContain('k8s:pod:kube-system/coredns')
    expect(f('', 'all', false, false)).not.toContain('proc:4:0')
    expect(f('', 'all', false, true)).toContain('k8s:pod:kube-system/coredns')
    expect(f('', 'all', false, true)).not.toContain('proc:4:0')
    expect(f('', 'all', true, false)).toContain('proc:4:0')
    expect(f('', 'all', true, false)).not.toContain('k8s:pod:kube-system/coredns')
  })

  it('isSystem só para namespaces de sistema', () => {
    expect(isSystem(items[2])).toBe(true)
    expect(isSystem(items[5])).toBe(false)
  })

  it('chipCount respeita os ocultos', () => {
    expect(chipCount(items, 'procs', false, false)).toBe(2)
  })
})
