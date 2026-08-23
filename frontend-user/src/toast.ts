import { reactive } from 'vue'

export type ToastItem = { id: number; text: string; kind: 'ok' | 'err' | 'warn' }

export const toasts = reactive<ToastItem[]>([])
let seq = 1

export function toast(text: string, kind: ToastItem['kind'] = 'ok') {
  const id = seq++
  toasts.push({ id, text, kind })
  setTimeout(() => dismiss(id), 5000)
}

export function dismiss(id: number) {
  const i = toasts.findIndex((t) => t.id === id)
  if (i >= 0) toasts.splice(i, 1)
}
