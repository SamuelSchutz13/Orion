<script setup lang="ts">
// Painel da direita: cabeçalho do item, detalhes e, se houver, o log ao vivo.
import { computed } from 'vue'
import type { Action, Item } from '@/lib/api'
import { actionLabel, dangerous, kindLabel, statusDot, statusTitle } from '@/lib/format'
import { Button } from '@/components/ui/button'
import type { Hist } from '@/lib/history'
import LogPanel from './LogPanel.vue'
import Sparkline from './Sparkline.vue'

const props = defineProps<{ item: Item | null; hist?: Hist }>()
const pct = (v: number) => v.toFixed(1) + '%'
const mb = (v: number) => (v >= 1024 ? (v / 1024).toFixed(1) + ' GB' : v.toFixed(1) + ' MB')
const emit = defineEmits<{ act: [item: Item, action: Action] }>()

const meta = computed(() => {
  const i = props.item
  if (!i) return []
  const d = i.detail ?? {}
  return ([
    ['Portas', i.ports.map(p => ':' + p).join(' ')],
    ['PID', i.pid ? String(i.pid) : ''],
    ['Namespace', i.namespace],
    ['Imagem', i.image],
    ...(props.hist ? [] : [['CPU', d.cpu], ['Memória', d.mem]]), ['Usuário', d.user],
    ['Filhos', d.children], ['Reinícios', d.restarts], ['Réplicas', d.replicas], ['Código de saída', d.exitCode],
    ['Estado', d.status ?? d.phase], ['Iniciado', d.started],
  ] as [string, string | undefined][]).filter(([, v]) => v)
})
const longLines = computed(() => {
  const d = props.item?.detail ?? {}
  return [d.path, d.cmdline && d.cmdline !== props.item?.name ? d.cmdline : ''].filter(Boolean) as string[]
})
</script>

<template>
  <section class="flex min-h-0 flex-col bg-panel" aria-label="Detalhes e logs">
    <div v-if="!item" class="m-[18px] rounded-[10px] border border-dashed p-4 text-muted-foreground">Selecione um item da lista.</div>
    <template v-else>
      <div class="flex min-w-0 items-center gap-2.5 border-b px-4 py-3.5 sm:px-[18px]">
        <span class="inline-block size-2 shrink-0 rounded-full" :class="statusDot[item.status]" :title="statusTitle[item.status]" />
        <h2 class="truncate text-[15px] font-semibold" :title="item.name">{{ item.name }}</h2>
        <span class="tag" :class="`t-${item.kind}`">{{ kindLabel[item.kind] }}</span>
        <div class="flex-1" />
        <div class="flex shrink-0 gap-1">
          <Button v-for="a in item.actions" :key="a" size="sm" :variant="dangerous(a) ? 'destructive' : 'outline'" @click="emit('act', item, a)">{{ actionLabel(a, item) }}</Button>
        </div>
      </div>
      <div v-if="meta.length || longLines.length" class="flex flex-wrap gap-x-4 gap-y-1.5 border-b px-4 py-2.5 text-xs text-muted-foreground sm:px-[18px]">
        <span v-for="[k, v] in meta" :key="k">{{ k }} <b class="font-mono font-medium text-foreground">{{ v }}</b></span>
        <span v-for="l in longLines" :key="l" class="basis-full truncate font-mono" :title="l">{{ l }}</span>
      </div>

      <div v-if="hist" class="grid grid-cols-2 gap-2.5 border-b px-4 py-2.5 sm:px-[18px]">
        <div class="tile px-2.5 pt-2 pb-1.5">
          <div class="tile-lbl">CPU <span>60 s</span></div>
          <div class="tile-val text-[15px]">{{ pct(hist.cpu.at(-1) ?? 0) }}</div>
          <Sparkline :data="hist.cpu" :fmt="pct" />
        </div>
        <div class="tile px-2.5 pt-2 pb-1.5">
          <div class="tile-lbl">Memória <span>60 s</span></div>
          <div class="tile-val text-[15px]">{{ mb(hist.mem.at(-1) ?? 0) }}</div>
          <Sparkline :data="hist.mem" :fmt="mb" />
        </div>
      </div>

      <LogPanel v-if="item.hasLogs" :item-id="item.id" />
      <div v-else class="m-[18px] rounded-[10px] border border-dashed p-4 text-muted-foreground">
        <template v-if="item.kind === 'service'">Services não têm log próprio. Veja os logs dos pods por trás dele.</template>
        <template v-else-if="item.kind === 'deployment'">Deployments não têm log próprio. Veja os logs dos pods dele.</template>
        <template v-else>
          O Windows não deixa ler a saída de um processo que o Orion não iniciou.<br /><br />
          Para ver os logs aqui, rode com
          <code class="rounded border bg-card px-1.5 py-px break-all text-foreground">orion run {{ item.detail?.cmdline || '…' }}</code>
          ou acompanhe um arquivo <code class="rounded border bg-card px-1.5 py-px text-foreground">.log</code>.
        </template>
      </div>
    </template>
  </section>
</template>
