import type { Action, Item, Kind, Status } from './api'

export const kindLabel: Record<Kind, string> = {
  proc: 'processo', run: 'orion run', docker: 'docker', pod: 'pod', service: 'service', deployment: 'deployment', logfile: 'arquivo',
}

export const statusTitle: Record<Status, string> = { ok: 'rodando', warn: 'com problema', down: 'parado' }
export const statusDot: Record<Status, string> = { ok: 'bg-ok', warn: 'bg-warn', down: 'bg-down' }

export const actionLabel = (a: Action, i: Item): string =>
  ({ kill: 'encerrar', start: 'iniciar', stop: 'parar', restart: 'reiniciar', delete: 'apagar', dismiss: i.kind === 'run' ? 'dispensar' : 'remover' })[a]

export const dangerous = (a: Action) => a === 'kill' || a === 'stop' || a === 'delete'

// Linha de baixo de cada item da lista.
export function subLine(i: Item): string {
  const d = i.detail ?? {}
  switch (i.kind) {
    case 'proc': return [i.pid != null && `PID ${i.pid}`, d.cmdline && d.cmdline !== i.name && d.cmdline].filter(Boolean).join(' · ')
    case 'run': return [`PID ${i.pid}`, d.exitCode != null && `saiu com código ${d.exitCode}`].filter(Boolean).join(' · ')
    case 'docker': return [i.image, d.status].filter(Boolean).join(' · ')
    case 'pod': return [i.namespace, Number(d.restarts) > 0 && `reiniciou ${d.restarts}x`, d.reason].filter(Boolean).join(' · ')
    case 'service': return [i.namespace, d.type, d.clusterIP].filter(Boolean).join(' · ')
    case 'deployment': return [i.namespace, d.replicas && `${d.replicas} réplicas`].filter(Boolean).join(' · ')
    case 'logfile': return d.path ?? ''
  }
}

export const portsText = (i: Item) => i.ports.map(p => ':' + p).join(' ')

export function confirmText(a: Action, i: Item, children = 0): { title: string; body: string } {
  const ports = i.ports.length ? ` A porta ${i.ports.map(p => ':' + p).join(', ')} será liberada.` : ''
  switch (a) {
    case 'kill': return {
      title: `Encerrar ${i.name}?`,
      body: `PID ${i.pid}${children ? ` e ${children} processo(s) filho(s) serão encerrados.` : ' será encerrado.'}${ports}`,
    }
    case 'stop': return { title: `Parar ${i.name}?`, body: `O container será parado. Os dados do volume são mantidos.${ports}` }
    case 'restart': return { title: `Reiniciar ${i.name}?`, body: 'O container será parado e iniciado de novo.' }
    case 'start': return { title: `Iniciar ${i.name}?`, body: 'O container será iniciado.' }
    case 'delete': return { title: `Apagar o pod ${i.name}?`, body: 'Se ele pertence a um deployment, um pod novo é criado no lugar.' }
    case 'dismiss': return i.kind === 'run'
      ? { title: `Dispensar ${i.name}?`, body: 'A linha e o log somem do painel.' }
      : { title: `Parar de acompanhar ${i.name}?`, body: 'O arquivo não é apagado, só sai da lista.' }
  }
}

export const fmtBytes = (b: number) =>
  b >= 1 << 30 ? (b / (1 << 30)).toFixed(1) + ' GB' : (b / (1 << 20)).toFixed(0) + ' MB'
