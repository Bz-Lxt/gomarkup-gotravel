<script setup lang="ts">
import { onMounted } from 'vue'
import { useAuth } from './store'
import { toasts, dismiss } from './toast'

const auth = useAuth()
onMounted(() => { auth.boot() })
</script>

<template>
  <div class="grain min-h-full">
    <router-view />
    <div class="fixed right-4 top-4 z-[4000] flex w-80 flex-col gap-2">
      <div
        v-for="t in toasts"
        :key="t.id"
        class="flex items-start justify-between rounded-xl border px-3 py-2 text-sm shadow-lg backdrop-blur"
        :class="t.kind === 'err' ? 'border-clay/50 bg-ink/90 text-paper' : t.kind === 'warn' ? 'border-lantern/50 bg-ink/90' : 'border-moss/40 bg-ink/90'"
      >
        <span>{{ t.text }}</span>
        <button class="ml-2 text-paper/50" @click="dismiss(t.id)">×</button>
      </div>
    </div>
  </div>
</template>
