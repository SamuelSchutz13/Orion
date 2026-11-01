<script setup lang="ts">
// Mini gráfico dos últimos ~60 s (30 amostras de 2 s), só na memória do navegador.
import { computed, ref } from 'vue'

const props = defineProps<{ data: number[]; fmt: (v: number) => string; max?: number }>()
const N = 30, W = 100, H = 30
const hover = ref<number | null>(null)

const scale = computed(() => {
  const d = props.data.length ? props.data : [0]
  let lo = Math.min(...d), hi = props.max ?? Math.max(...d)
  const pad = Math.max((hi - lo) * 0.15, Math.abs(hi) * 0.01, 1e-3)
  lo -= pad; hi += pad
  return (v: number) => H - ((v - lo) / (hi - lo)) * H
})
const step = W / (N - 1)
const off = computed(() => (N - props.data.length) * step)
const pts = computed(() => props.data.map((v, k) => [off.value + k * step, scale.value(v)] as const))
const line = computed(() => pts.value.map(p => p.join(',')).join(' '))

function move(e: MouseEvent) {
  const r = (e.currentTarget as SVGElement).getBoundingClientRect()
  const x = ((e.clientX - r.left) / r.width) * W
  hover.value = Math.max(0, Math.min(props.data.length - 1, Math.round((x - off.value) / step)))
}
const tip = computed(() => {
  if (hover.value == null || !props.data.length) return ''
  const ago = (props.data.length - 1 - hover.value) * 2
  return props.fmt(props.data[hover.value]) + (ago ? ` · há ${ago}s` : ' · agora')
})
</script>

<template>
  <div class="relative">
    <svg :viewBox="`0 0 ${W} ${H}`" preserveAspectRatio="none" class="mt-1 block h-[30px] w-full cursor-crosshair overflow-visible"
      role="img" aria-label="Últimos 60 segundos" @mousemove="move" @mouseleave="hover = null">
      <template v-if="pts.length > 1">
        <polygon :points="`${off},${H} ${line} ${W},${H}`" class="fill-brand opacity-[.12]" />
        <polyline :points="line" class="fill-none stroke-brand" stroke-width="2" vector-effect="non-scaling-stroke" stroke-linejoin="round" stroke-linecap="round" />
      </template>
      <template v-if="hover != null && pts[hover]">
        <line :x1="pts[hover][0]" :x2="pts[hover][0]" y1="0" :y2="H" class="stroke-faint" stroke-width="1" vector-effect="non-scaling-stroke" />
        <circle :cx="pts[hover][0]" :cy="pts[hover][1]" r="2.5" class="fill-brand stroke-card" stroke-width="2" vector-effect="non-scaling-stroke" />
      </template>
    </svg>
    <div v-if="tip" class="pointer-events-none absolute -top-1 z-10 -translate-x-1/2 -translate-y-full whitespace-nowrap rounded-md bg-foreground px-1.5 py-0.5 text-[11px] text-background"
      :style="{ left: (pts[hover!]?.[0] ?? 0) + '%' }">{{ tip }}</div>
  </div>
</template>
