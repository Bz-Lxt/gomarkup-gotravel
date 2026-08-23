<script setup lang="ts">
import { useAuth } from '../store'
import { useRouter } from 'vue-router'

defineProps<{ title: string; sub?: string }>()
const auth = useAuth()
const router = useRouter()
function out() {
  auth.logout()
  router.push('/login')
}
</script>
<template>
  <div class="flex min-h-full flex-col">
    <header class="flex w-full items-center justify-between border-b border-paper/10 px-5 py-3">
      <div>
        <div class="font-display text-2xl tracking-wide">{{ title }}</div>
        <div v-if="sub" class="text-xs text-paper/55">{{ sub }}</div>
      </div>
      <div class="flex items-center gap-3 text-sm">
        <router-link to="/" class="rounded-full border border-paper/15 px-3 py-1 hover:border-lantern">队伍</router-link>
        <span class="hidden text-paper/60 sm:inline">{{ auth.user?.nickname }}</span>
        <button class="rounded-full bg-moss px-3 py-1 text-paper" @click="out">退出</button>
      </div>
    </header>
    <main class="w-full flex-1">
      <slot />
    </main>
  </div>
</template>
