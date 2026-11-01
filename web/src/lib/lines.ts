import type { Line } from './api'

export const MAX_LINES = 5000

// Mantém só as últimas `cap` linhas, para o DOM não crescer sem limite.
export function appendCapped(buf: Line[], add: Line[], cap = MAX_LINES): Line[] {
  const all = buf.concat(add)
  return all.length > cap ? all.slice(all.length - cap) : all
}

export type Level = 'err' | 'warn' | 'ok' | 'sys' | 'info'

const errRe = /\b(error|erro|panic|fatal|exception|failed|falhou|traceback)\b/i
const warnRe = /\b(warn|warning|aviso|deprecated)\b/i
const okRe = /\b(ready|listening|started|done|success|compiled)\b/i

export function levelOf(l: Line): Level {
  if (l.stream === 'sys') return 'sys'
  if (errRe.test(l.text)) return 'err'
  if (warnRe.test(l.text)) return 'warn'
  if (l.stream === 'err') return 'err'
  if (okRe.test(l.text)) return 'ok'
  return 'info'
}
