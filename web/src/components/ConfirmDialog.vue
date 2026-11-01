<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { type Action, type Item, api } from '@/lib/api'
import { actionLabel, confirmText, dangerous } from '@/lib/format'
import {
  AlertDialog, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'

const props = defineProps<{ target: { item: Item; action: Action } | null }>()
const emit = defineEmits<{ close: []; done: [msg: string] }>()

const children = ref(0)
const error = ref('')
const busy = ref(false)
const loading = ref(false)

// Para encerrar, pergunta ao servidor quantos filhos serão levados junto (e se o PID ainda é o mesmo).
watch(() => props.target, async t => {
  children.value = 0; error.value = ''; busy.value = false
  if (t?.action !== 'kill') return
  loading.value = true
  try { children.value = (await api.preview(t.item.id)).children } catch (e) { error.value = (e as Error).message }
  loading.value = false
})

const text = computed(() => props.target && confirmText(props.target.action, props.target.item, children.value))

async function confirm() {
  const t = props.target!
  busy.value = true; error.value = ''
  try {
    await api.act(t.item.id, t.action)
    emit('done', `${actionLabel(t.action, t.item)}: ${t.item.name} ✓`)
  } catch (e) {
    error.value = (e as Error).message
  }
  busy.value = false
}
</script>

<template>
  <AlertDialog :open="!!target" @update:open="o => !o && emit('close')">
    <AlertDialogContent v-if="target && text">
      <AlertDialogHeader>
        <AlertDialogTitle class="line-clamp-3 [overflow-wrap:anywhere]" :title="text.title">{{ text.title }}</AlertDialogTitle>
        <AlertDialogDescription class="[overflow-wrap:anywhere]">{{ loading ? 'Verificando o processo…' : text.body }}</AlertDialogDescription>
      </AlertDialogHeader>
      <p v-if="error" class="rounded-md border border-danger-bd bg-danger-bg px-3 py-2 text-sm text-danger-fg" role="alert">{{ error }}</p>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancelar</AlertDialogCancel>
        <Button :variant="dangerous(target.action) ? 'destructive' : 'default'" :disabled="busy || loading" @click="confirm">
          {{ actionLabel(target.action, target.item).replace(/^./, c => c.toUpperCase()) }}
        </Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
