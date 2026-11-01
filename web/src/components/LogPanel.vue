<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { type Line, streamLogs } from '@/lib/api'
import { appendCapped, levelOf } from '@/lib/lines'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const props = defineProps<{ itemId: string }>()

const lines = ref<Line[]>([])
const paused = ref(false)
const filter = ref('')
const follow = ref(true)
const box = ref<HTMLElement | null>(null)

// Linhas chegam em rajadas; juntamos tudo e desenhamos uma vez por frame.
let pending: Line[] = []
let frame = 0
let close: (() => void) | null = null

function flush() {
  frame = 0
  if (paused.value || !pending.length) return
  lines.value = appendCapped(lines.value, pending)
  pending = []
  if (follow.value) nextTick(() => box.value && (box.value.scrollTop = box.value.scrollHeight))
}
function onLines(ls: Line[]) {
  pending = appendCapped(pending, ls) // pausado também fica limitado
  if (!frame) frame = requestAnimationFrame(flush)
}

watch(() => props.itemId, id => {
  close?.()
  lines.value = []; pending = []; filter.value = ''; paused.value = false; follow.value = true
  close = streamLogs(id, onLines)
}, { immediate: true })
onBeforeUnmount(() => { close?.(); cancelAnimationFrame(frame) })

function togglePause() {
  paused.value = !paused.value
  if (!paused.value) { follow.value = true; flush() }
}
function onScroll() {
  const b = box.value!
  follow.value = b.scrollHeight - b.scrollTop - b.clientHeight < 24
}
function jump() { follow.value = true; nextTick(() => box.value && (box.value.scrollTop = box.value.scrollHeight)) }

const shown = computed(() => {
  const f = filter.value.toLowerCase()
  return f ? lines.value.filter(l => l.text.toLowerCase().includes(f)) : lines.value
})
const counts = computed(() => {
  let err = 0, warn = 0
  for (const l of lines.value) { const v = levelOf(l); if (v === 'err') err++; else if (v === 'warn') warn++ }
  return { err, warn }
})
// partes do texto com o trecho do filtro marcado (sem v-html)
function parts(text: string): { t: string; m: boolean }[] {
  const f = filter.value.toLowerCase()
  if (!f) return [{ t: text, m: false }]
  const out: { t: string; m: boolean }[] = []
  const low = text.toLowerCase()
  let i = 0
  for (let j = low.indexOf(f); j >= 0; j = low.indexOf(f, i)) {
    if (j > i) out.push({ t: text.slice(i, j), m: false })
    out.push({ t: text.slice(j, j + f.length), m: true })
    i = j + f.length
  }
  if (i < text.length) out.push({ t: text.slice(i), m: false })
  return out
}
const lvClass = { err: 'text-lv-err', warn: 'text-lv-warn', ok: 'text-lv-ok', sys: 'text-faint italic', info: '' }
const ts = (t: string) => new Date(t).toTimeString().slice(0, 8)
</script>

<template>
  <div class="flex items-center gap-1.5 px-4 py-2.5 sm:px-[18px]">
    <Input v-model="filter" type="search" aria-label="Filtrar log" placeholder="Filtrar log…" spellcheck="false" class="h-7 min-w-0 flex-1 text-xs" />
    <span class="text-[11.5px] whitespace-nowrap text-muted-foreground tabular-nums">
      {{ lines.length }} linhas · <b class="font-medium text-lv-err">{{ counts.err }} erros</b> · <span class="text-lv-warn">{{ counts.warn }} avisos</span>
    </span>
    <Button size="sm" variant="outline" :aria-pressed="paused" @click="togglePause">{{ paused ? '▶ continuar' : '❚❚ pausar' }}</Button>
    <Button size="sm" variant="outline" @click="lines = []">limpar</Button>
  </div>
  <div class="relative min-h-0 flex-1">
    <div ref="box" class="absolute inset-0 overflow-auto px-4 pt-1 pb-[18px] font-mono text-xs leading-[1.65] sm:px-[18px]" role="log" aria-live="off" @scroll="onScroll">
      <div v-for="l in shown" :key="l.seq" class="break-words whitespace-pre-wrap" :class="lvClass[levelOf(l)]">
        <span class="mr-2 text-faint select-none">{{ ts(l.time) }}</span><template v-for="(p, k) in parts(l.text)" :key="k"><mark v-if="p.m">{{ p.t }}</mark><template v-else>{{ p.t }}</template></template>
      </div>
      <div v-if="!shown.length" class="py-4 text-faint">{{ filter ? 'Nenhuma linha corresponde ao filtro.' : 'Sem linhas ainda. Aguardando saída…' }}</div>
    </div>
    <Button v-if="!follow" size="sm" variant="outline" class="absolute right-[18px] bottom-3.5 shadow-lg" @click="jump">↓ ir para o fim</Button>
  </div>
</template>
