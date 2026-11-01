import { describe, expect, it } from 'vitest'
import type { Line } from './api'
import { appendCapped, levelOf } from './lines'

const mk = (n: number, from = 0): Line[] =>
  Array.from({ length: n }, (_, i) => ({ seq: from + i, time: '', stream: 'out', text: String(from + i) }))

describe('appendCapped', () => {
  it('limita a 5000 e mantém as mais novas', () => {
    const r = appendCapped(mk(4999), mk(10, 4999))
    expect(r.length).toBe(5000)
    expect(r.at(-1)!.text).toBe('5008')
    expect(r[0].text).toBe('9')
  })
  it('respeita cap customizado', () => {
    expect(appendCapped(mk(3), mk(3, 3), 4).map(l => l.text)).toEqual(['2', '3', '4', '5'])
  })
})

describe('levelOf', () => {
  it('classifica linhas', () => {
    expect(levelOf({ seq: 1, time: '', stream: 'out', text: 'panic: nil pointer' })).toBe('err')
    expect(levelOf({ seq: 1, time: '', stream: 'out', text: 'WARNING memory' })).toBe('warn')
    expect(levelOf({ seq: 1, time: '', stream: 'err', text: 'npm notice' })).toBe('err')
    expect(levelOf({ seq: 1, time: '', stream: 'err', text: 'warning: slow query' })).toBe('warn')
    expect(levelOf({ seq: 1, time: '', stream: 'sys', text: 'terminou' })).toBe('sys')
    expect(levelOf({ seq: 1, time: '', stream: 'out', text: 'GET / 200' })).toBe('info')
  })
})
