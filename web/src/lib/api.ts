// Tipos espelhados de internal/inventory e internal/logs (Go).
export type Kind = 'proc' | 'docker' | 'pod' | 'service' | 'deployment' | 'logfile' | 'run'
export type Status = 'ok' | 'warn' | 'down'
export type Action = 'kill' | 'start' | 'stop' | 'restart' | 'delete' | 'dismiss'

export interface Item {
  id: string
  kind: Kind
  name: string
  status: Status
  ports: number[]
  pid?: number
  namespace?: string
  image?: string
  detail?: Record<string, string>
  hasLogs: boolean
  actions: Action[] | readonly Action[]
  hidden: boolean
}

export interface SourceState { ok: boolean; reason?: string }
export interface Machine { cpu: number; cores: number; memUsed: number; memTotal: number }
export interface Snapshot { items: Item[]; sources: Record<string, SourceState>; machine?: Machine }

export interface Line { seq: number; time: string; stream: 'out' | 'err' | 'sys'; text: string }
export interface PreviewInfo { name: string; pid: number; children: number }

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `erro HTTP ${res.status}`)
  }
  return res.status === 204 ? (undefined as T) : res.json()
}

export const api = {
  preview: (id: string) => call<PreviewInfo>('GET', `/api/actions/preview?id=${encodeURIComponent(id)}`),
  act: (id: string, action: Action) => call<void>('POST', '/api/actions', { id, action }),
  addLogFile: (path: string) => call<{ path: string }>('POST', '/api/logfiles', { path }),
}

// O EventSource reconecta sozinho; onState avisa a UI quando a conexão cai ou volta.
export function streamSnapshots(onSnap: (s: Snapshot) => void, onState: (live: boolean) => void): () => void {
  const es = new EventSource('/api/stream')
  es.addEventListener('snapshot', e => { onState(true); onSnap(JSON.parse((e as MessageEvent).data)) })
  es.onerror = () => onState(false)
  return () => es.close()
}

export function streamLogs(id: string, onLines: (l: Line[]) => void): () => void {
  const es = new EventSource(`/api/logs?id=${encodeURIComponent(id)}`)
  es.addEventListener('lines', e => onLines(JSON.parse((e as MessageEvent).data)))
  return () => es.close()
}
