<script setup lang="ts">
import { nextTick } from 'vue'
import type { Action, Item } from '@/lib/api'
import { actionLabel, dangerous, kindLabel, statusDot, statusTitle, subLine } from '@/lib/format'
import { Button } from '@/components/ui/button'

const props = defineProps<{ items: Item[]; selected: string | null; query: string; hiddenMatches: number }>()
const emit = defineEmits<{ select: [id: string]; act: [item: Item, action: Action]; showHidden: [] }>()

function key(e: KeyboardEvent, i: Item) {
  if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); emit('select', i.id) }
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    const idx = props.items.findIndex(x => x.id === i.id) + (e.key === 'ArrowDown' ? 1 : -1)
    const next = props.items[idx]
    if (!next) return
    emit('select', next.id)
    nextTick(() => document.querySelector<HTMLElement>(`tr[data-id="${CSS.escape(next.id)}"]`)?.focus())
  }
}
const cpuMem = (i: Item) => (i.detail?.cpu && i.detail?.mem ? `${i.detail.cpu} · ${i.detail.mem}` : '')
</script>

<template>
  <div class="min-h-0 flex-1 overflow-auto px-2.5 pb-3">
    <table class="w-full border-separate border-spacing-0">
      <thead>
        <tr class="[&>th]:sticky [&>th]:top-0 [&>th]:z-[1] [&>th]:border-b [&>th]:bg-background [&>th]:p-2 [&>th]:text-left [&>th]:text-xs [&>th]:font-medium [&>th]:text-muted-foreground">
          <th>Nome</th><th class="max-sm:hidden">Tipo</th><th>Portas</th><th class="max-sm:hidden">CPU · Mem</th><th><span class="sr-only">Ações</span></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="i in items" :key="i.id" :data-id="i.id" tabindex="0" :aria-selected="i.id === selected"
          class="group cursor-pointer outline-none [&>td]:border-b [&>td]:border-line [&>td]:px-2 [&>td]:py-[9px] [&>td]:whitespace-nowrap [&>td]:align-middle hover:[&>td]:bg-hover focus-visible:[&>td]:bg-hover"
          :class="i.id === selected && '[&>td]:bg-sel [&>td:first-child]:shadow-[inset_2px_0_0_var(--accent-strong)]'"
          @click="emit('select', i.id)" @keydown="key($event, i)">
          <td class="w-[42%] max-w-0 overflow-hidden">
            <span class="flex min-w-0 items-center gap-2">
              <span class="inline-block size-2 shrink-0 rounded-full" :class="statusDot[i.status]" :title="statusTitle[i.status]" />
              <span class="truncate font-medium" :class="i.hidden && 'text-muted-foreground'">{{ i.name }}</span>
            </span>
            <span class="block truncate pl-4 font-mono text-xs text-muted-foreground">{{ subLine(i) }}</span>
          </td>
          <td class="max-sm:hidden"><span class="tag" :class="`t-${i.kind}`">{{ kindLabel[i.kind] }}</span></td>
          <td class="font-mono text-[13px] whitespace-normal!">
            <span v-for="p in i.ports" :key="p" class="mr-1.5">:{{ p }}</span>
            <span v-if="!i.ports.length" class="text-xs text-muted-foreground">—</span>
          </td>
          <td class="font-mono text-xs text-muted-foreground tabular-nums max-sm:hidden">{{ cpuMem(i) }}</td>
          <td>
            <div class="flex justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100"
              :class="i.id === selected && 'opacity-100'">
              <Button v-for="a in i.actions" :key="a" size="xs" :variant="dangerous(a) ? 'destructive' : 'outline'"
                @click.stop="emit('act', i, a)">{{ actionLabel(a, i) }}</Button>
            </div>
          </td>
        </tr>
        <tr v-if="!items.length">
          <td colspan="5" class="px-4 py-10 text-center whitespace-normal text-muted-foreground">
            Nada encontrado<template v-if="query"> para “{{ query }}”</template>.
          </td>
        </tr>
      </tbody>
    </table>
    <div v-if="hiddenMatches" class="px-2 py-2.5 text-xs text-muted-foreground">
      {{ hiddenMatches }} item(ns) oculto(s){{ query ? ' correspondem à busca' : '' }}.
      <button class="cursor-pointer text-brand underline underline-offset-2" @click="emit('showHidden')">Mostrar</button>
    </div>
  </div>
</template>
