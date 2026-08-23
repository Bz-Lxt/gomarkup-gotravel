<script setup lang="ts">
defineProps<{ title: string }>()
const emit = defineEmits<{ close: [] }>()
</script>
<template>
  <div class="fixed inset-0 z-[3000] flex items-center justify-center bg-black/55 p-4" @click.self="emit('close')">
    <div class="w-full max-w-md rounded-2xl border border-paper/10 bg-dusk p-5 shadow-2xl">
      <div class="mb-4 flex items-center justify-between">
        <h3 class="font-display text-xl">{{ title }}</h3>
        <button class="text-paper/50" @click="emit('close')">×</button>
      </div>
      <slot />
    </div>
  </div>
</template>
