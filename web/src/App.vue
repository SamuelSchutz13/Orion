<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Moon, Plus, Search, Sun } from '@lucide/vue'
import { type Action, type Item, type Snapshot, api, streamSnapshots } from '@/lib/api'
import { type Chip, chipCount, chips, filterItems, isSystem } from '@/lib/filter'
import { type Hist, updateHistory } from '@/lib/history'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import ItemList from '@/components/ItemList.vue'
import SidePanel from '@/components/SidePanel.vue'
import StatTiles from '@/components/StatTiles.vue'

const snap = ref<Snapshot>({ items: [], sources: {} })
const live = ref(false)
const loaded = ref(false)
const q = ref('')
const chip = ref<Chip>('all')
const showAll = ref(false)
const showSystem = ref(false)
const selId = ref<string | null>(null)
const target = ref<{ item: Item; action: Action } | null>(null)
const cpuHist = ref<number[]>([])
const memHist = ref<number[]>([])
const itemHist = ref(new Map<string, Hist>())
const search = ref<InstanceType<typeof Input> | null>(null)
const host = location.host

// Ordem estável: a lista não pula a cada atualização.
const kindOrder: Record<string, number> = { run: 0, proc: 1, docker: 2, pod: 3, deployment: 4, service: 5, logfile: 6 }
const sorted = computed(() => [...snap.value.items].sort((a, b) =>
  Number(a.hidden) - Number(b.hidden) || kindOrder[a.kind] - kindOrder[b.kind] ||
  a.name.localeCompare(b.name, 'pt-BR', { sensitivity: 'base' }) || (a.pid ?? 0) - (b.pid ?? 0) || a.id.localeCompare(b.id)))
const rows = computed(() => filterItems(sorted.value, q.value, chip.value, showAll.value, showSystem.value))
const hiddenMatches = computed(() => filterItems(sorted.value, q.value, chip.value, true, true).length - rows.value.length)
const selected = computed(() => snap.value.items.find(i => i.id === selId.value) ?? null)
const hiddenProcs = computed(() => snap.value.items.filter(i => i.hidden && !isSystem(i)).length)
const hiddenSys = computed(() => snap.value.items.filter(isSystem).length)
const k8sContext = computed(() => snap.value.items.find(i => i.detail?.context)?.detail?.context)

function onSnap(s: Snapshot) {
  snap.value = s
  loaded.value = true
  itemHist.value = updateHistory(itemHist.value, s.items)
  if (s.machine) {
    cpuHist.value = [...cpuHist.value, s.machine.cpu].slice(-30)
    memHist.value = [...memHist.value, s.machine.memUsed / 2 ** 30].slice(-30)
  }
  if (!selId.value && rows.value.length) selId.value = rows.value[0].id
}

// toast
const toast = ref<{ msg: string; err: boolean } | null>(null)
let tt = 0
function showToast(msg: string, err = false) {
  toast.value = { msg, err }
  clearTimeout(tt)
  tt = window.setTimeout(() => (toast.value = null), 2600)
}
function done(msg: string) { target.value = null; showToast(msg) }

// arquivos .log
const adding = ref(false)
const newPath = ref('')
async function addFile() {
  try {
    const { path } = await api.addLogFile(newPath.value)
    newPath.value = ''; adding.value = false
    selId.value = 'file:' + path
    showToast('Acompanhando ' + path)
  } catch (e) {
    showToast((e as Error).message, true)
  }
}

// tema
const dark = ref(document.documentElement.classList.contains('dark'))
function toggleTheme() {
  dark.value = !dark.value
  document.documentElement.classList.toggle('dark', dark.value)
  try { localStorage.setItem('orion-theme', dark.value ? 'dark' : 'light') } catch { /* sem storage: só nesta aba */ }
}

function onKey(e: KeyboardEvent) {
  if (e.key === '/' && !/INPUT|TEXTAREA/.test((document.activeElement as HTMLElement)?.tagName)) {
    e.preventDefault()
    ;(search.value?.$el as HTMLInputElement | undefined)?.focus()
  }
}
let stop: () => void = () => {}
onMounted(() => {
  stop = streamSnapshots(onSnap, v => (live.value = v))
  document.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => { stop(); document.removeEventListener('keydown', onKey) })
</script>

<template>
  <div class="flex h-full flex-col max-[900px]:h-auto max-[900px]:min-h-full">
    <header class="flex shrink-0 items-center gap-3 border-b px-4 py-2.5 sm:px-[18px]">
      <div class="flex items-center gap-2.5 font-semibold tracking-tight">
        <img src="/favicon.svg" alt="" class="size-[26px]" />Orion Logger
      </div>
      <span class="rounded-full border px-2 py-0.5 font-mono text-[11px] whitespace-nowrap text-muted-foreground max-[900px]:hidden">{{ host }}</span>
      <div class="flex-1" />
      <span v-if="k8sContext" class="rounded-full border px-2 py-0.5 text-[11px] whitespace-nowrap text-muted-foreground max-[900px]:hidden">k8s: {{ k8sContext }}</span>
      <span class="flex items-center gap-1.5 text-xs whitespace-nowrap text-muted-foreground" title="Atualiza a cada 2 segundos">
        <i v-if="live" class="live-dot" /><i v-else class="size-[7px] rounded-full bg-warn" />{{ live ? 'ao vivo' : loaded ? 'reconectando…' : 'conectando…' }}
      </span>
      <Button variant="outline" size="icon" :aria-label="dark ? 'Mudar para tema claro' : 'Mudar para tema escuro'" :title="dark ? 'Mudar para tema claro' : 'Mudar para tema escuro'" @click="toggleTheme">
        <Sun v-if="dark" /><Moon v-else />
      </Button>
    </header>

    <StatTiles :snap="snap" :cpu-hist="cpuHist" :mem-hist="memHist" />

    <main class="grid min-h-0 flex-1 grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)] max-[900px]:grid-cols-1 max-[900px]:grid-rows-[minmax(420px,1.2fr)_minmax(360px,1fr)]">
      <section class="flex min-h-0 flex-col border-r max-[900px]:border-r-0 max-[900px]:border-b" aria-label="Itens em execução">
        <div class="flex flex-col gap-2.5 px-4 pt-3 pb-2.5 sm:px-[18px]">
          <div class="relative">
            <Search class="pointer-events-none absolute top-1/2 left-[11px] size-[15px] -translate-y-1/2 text-muted-foreground" />
            <Input ref="search" v-model="q" type="search" aria-label="Buscar" placeholder="Buscar porta, processo, container, pod…" autocomplete="off" spellcheck="false" class="h-9 pr-10 pl-[34px]" />
            <kbd class="absolute top-1/2 right-2.5 -translate-y-1/2 rounded border px-1.5 text-[11px] text-muted-foreground" aria-hidden="true">/</kbd>
          </div>
          <div class="flex flex-wrap items-center gap-1.5">
            <div class="flex flex-wrap gap-1.5" role="tablist" aria-label="Filtrar por tipo">
              <button v-for="c in chips" :key="c.key" role="tab" :aria-selected="c.key === chip"
                class="flex cursor-pointer items-center gap-1.5 rounded-full border px-[11px] py-1 text-xs transition-colors"
                :class="c.key === chip ? 'border-foreground bg-foreground text-background' : 'text-muted-foreground hover:bg-hover hover:text-foreground'"
                :title="c.source && !snap.sources[c.source]?.ok ? snap.sources[c.source]?.reason : undefined"
                @click="chip = c.key">
                {{ c.label }}
                <b v-if="c.source && snap.sources[c.source] && !snap.sources[c.source].ok" class="font-medium opacity-70">indisponível</b>
                <b v-else class="font-medium tabular-nums opacity-70">{{ chipCount(snap.items, c.key, showAll, showSystem) }}</b>
              </button>
            </div>
            <div class="ml-auto flex flex-wrap items-center gap-3 text-xs text-muted-foreground max-[900px]:ml-0">
              <label class="flex cursor-pointer items-center gap-2 whitespace-nowrap select-none">
                <Switch v-model="showAll" size="sm" />todos os processos <span v-if="!showAll" class="tabular-nums">(+{{ hiddenProcs }})</span>
              </label>
              <label v-if="hiddenSys" class="flex cursor-pointer items-center gap-2 whitespace-nowrap select-none">
                <Switch v-model="showSystem" size="sm" />namespaces de sistema
              </label>
              <Button size="xs" variant="outline" @click="adding = !adding"><Plus />arquivo .log</Button>
            </div>
          </div>
          <form v-if="adding" class="flex gap-1.5" @submit.prevent="addFile">
            <Input v-model="newPath" autofocus placeholder="C:\caminho\para\app.log" aria-label="Caminho do arquivo .log" spellcheck="false" class="h-8 flex-1 font-mono text-xs" />
            <Button size="sm" type="submit" :disabled="!newPath.trim()">Acompanhar</Button>
            <Button size="sm" variant="ghost" type="button" @click="adding = false">Cancelar</Button>
          </form>
        </div>
        <ItemList :items="rows" :selected="selId" :query="q" :hidden-matches="hiddenMatches"
          @select="id => (selId = id)" @act="(item, action) => (target = { item, action })"
          @show-hidden="showAll = true; showSystem = true" />
      </section>
      <SidePanel :item="selected" :hist="selId ? itemHist.get(selId) : undefined" @act="(item, action) => (target = { item, action })" />
    </main>

    <ConfirmDialog :target="target" @close="target = null" @done="done" />
    <div v-if="toast" role="status" aria-live="polite"
      class="fixed right-[18px] bottom-[18px] z-50 max-w-[calc(100%-36px)] rounded-[10px] border bg-card px-3.5 py-2.5 shadow-lg"
      :class="toast.err && 'border-danger-bd text-danger-fg'">{{ toast.msg }}</div>
  </div>
</template>
