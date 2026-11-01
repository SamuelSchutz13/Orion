<script setup lang="ts">
import { computed } from 'vue'
import type { Snapshot } from '@/lib/api'
import { fmtBytes } from '@/lib/format'
import Sparkline from './Sparkline.vue'

const props = defineProps<{ snap: Snapshot; cpuHist: number[]; memHist: number[] }>()

const m = computed(() => props.snap.machine)
const items = computed(() => props.snap.items)
const src = (k: string) => props.snap.sources[k]

const ports = computed(() => {
  const dev = new Set<number>(), all = new Set<number>()
  for (const i of items.value) for (const p of i.ports) { all.add(p); if (!i.hidden) dev.add(p) }
  return { all: all.size, dev: [...dev].sort((a, b) => a - b) }
})
const docker = computed(() => {
  const c = items.value.filter(i => i.kind === 'docker')
  const run = c.filter(i => i.status === 'ok').length
  return { total: c.length, run, stopped: c.length - run }
})
const pods = computed(() => {
  const p = items.value.filter(i => i.kind === 'pod' && !i.hidden)
  return {
    total: p.length,
    ready: p.filter(i => i.status === 'ok').length,
    restarts: p.reduce((s, i) => s + Number(i.detail?.restarts ?? 0), 0),
    services: items.value.filter(i => i.kind === 'service' && !i.hidden).length,
    namespaces: new Set(items.value.filter(i => i.namespace && !i.hidden).map(i => i.namespace)).size,
    context: items.value.find(i => i.detail?.context)?.detail?.context,
  }
})
const logs = computed(() => ({
  running: items.value.filter(i => i.kind === 'run' && i.status === 'ok').length,
  runs: items.value.filter(i => i.kind === 'run').length,
  files: items.value.filter(i => i.kind === 'logfile').length,
}))
const pct = (v: number) => v.toFixed(1) + '%'
const gb = (v: number) => v.toFixed(1) + ' GB'
</script>

<template>
  <section class="grid shrink-0 grid-cols-2 gap-2.5 border-b px-4 py-3 sm:px-[18px] md:grid-cols-3 xl:grid-cols-6" aria-label="Resumo da máquina">
    <div class="tile">
      <div class="tile-lbl">CPU <span v-if="m">{{ m.cores }} núcleos</span></div>
      <div class="tile-val">{{ m ? pct(m.cpu) : '—' }}</div>
      <Sparkline :data="cpuHist" :fmt="pct" />
    </div>
    <div class="tile">
      <div class="tile-lbl">Memória <span v-if="m" class="tabular-nums">{{ ((m.memUsed / m.memTotal) * 100).toFixed(0) }}%</span></div>
      <div class="tile-val">{{ m ? (m.memUsed / 2 ** 30).toFixed(1) : '—' }}<small v-if="m">/ {{ fmtBytes(m.memTotal) }}</small></div>
      <Sparkline :data="memHist" :fmt="gb" />
    </div>
    <div class="tile">
      <div class="tile-lbl">Portas abertas</div>
      <div class="tile-val">{{ ports.all }}</div>
      <div class="tile-det">{{ ports.dev.length }} visíveis · {{ ports.all - ports.dev.length }} ocultas</div>
      <div class="tile-det font-mono">{{ ports.dev.slice(0, 4).map(p => ':' + p).join(' ') }}{{ ports.dev.length > 4 ? ' …' : '' }}</div>
    </div>
    <div class="tile">
      <div class="tile-lbl">Containers</div>
      <template v-if="src('docker')?.ok">
        <div class="tile-val">{{ docker.run }}<small>/ {{ docker.total }} rodando</small></div>
        <div class="mt-2 flex h-1.5 gap-0.5 overflow-hidden rounded-full bg-track" role="img" :aria-label="`${docker.run} rodando, ${docker.stopped} parados`">
          <span v-if="docker.run" class="block h-full rounded-full bg-ok" :style="{ flex: docker.run }" />
          <span v-if="docker.stopped" class="block h-full rounded-full bg-down" :style="{ flex: docker.stopped }" />
        </div>
      </template>
      <template v-else>
        <div class="tile-val text-muted-foreground">indisponível</div>
        <div class="tile-det" :title="src('docker')?.reason">{{ src('docker')?.reason ?? 'carregando…' }}</div>
      </template>
    </div>
    <div class="tile">
      <div class="tile-lbl">Pods <span v-if="pods.context" class="truncate">{{ pods.context }}</span></div>
      <template v-if="src('k8s')?.ok">
        <div class="tile-val">{{ pods.ready }}<small>/ {{ pods.total }} prontos</small></div>
        <div class="tile-det">{{ pods.restarts ? `${pods.restarts} reinícios` : 'sem reinícios' }}</div>
        <div class="tile-det">{{ pods.services }} service · {{ pods.namespaces }} namespace(s)</div>
      </template>
      <template v-else>
        <div class="tile-val text-muted-foreground">indisponível</div>
        <div class="tile-det" :title="src('k8s')?.reason">{{ src('k8s')?.reason ?? 'carregando…' }}</div>
      </template>
    </div>
    <div class="tile">
      <div class="tile-lbl">Logs</div>
      <div class="tile-val">{{ logs.runs + logs.files }}<small>streams</small></div>
      <div class="tile-det">{{ logs.running }} orion run ativo(s)</div>
      <div class="tile-det">{{ logs.files }} arquivo(s) .log</div>
    </div>
  </section>
</template>
