import type { Item } from './api'

export type Chip = 'all' | 'ports' | 'procs' | 'docker' | 'k8s' | 'logs'

export const chips: { key: Chip; label: string; source?: string }[] = [
  { key: 'all', label: 'Tudo' },
  { key: 'ports', label: 'Portas' },
  { key: 'procs', label: 'Processos', source: 'procs' },
  { key: 'docker', label: 'Docker', source: 'docker' },
  { key: 'k8s', label: 'K8s', source: 'k8s' },
  { key: 'logs', label: 'Logs' },
]

const inChip: Record<Chip, (i: Item) => boolean> = {
  all: () => true,
  ports: i => i.ports.length > 0,
  procs: i => i.kind === 'proc' || i.kind === 'run',
  docker: i => i.kind === 'docker',
  k8s: i => i.kind === 'pod' || i.kind === 'service' || i.kind === 'deployment',
  logs: i => i.kind === 'logfile' || i.kind === 'run',
}

// Itens ocultos com namespace são de namespaces de sistema do Kubernetes; os outros são processos sem interesse.
export const isSystem = (i: Item) => i.hidden && !!i.namespace

const visible = (i: Item, showAll: boolean, showSystem: boolean) =>
  !i.hidden || (isSystem(i) ? showSystem : showAll)

const haystack = (i: Item) =>
  [i.name, i.namespace, i.image, i.pid, ...i.ports, i.detail?.path, i.detail?.cmdline].filter(v => v != null).join(' ').toLowerCase()

export function filterItems(items: Item[], q: string, chip: Chip, showAll: boolean, showSystem: boolean): Item[] {
  const needle = q.trim().toLowerCase()
  return items.filter(i => visible(i, showAll, showSystem) && inChip[chip](i) && (!needle || haystack(i).includes(needle)))
}

export const chipCount = (items: Item[], chip: Chip, showAll: boolean, showSystem: boolean) =>
  items.filter(i => visible(i, showAll, showSystem) && inChip[chip](i)).length
