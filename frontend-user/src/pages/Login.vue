<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../store'
import { toast } from '../toast'

const tab = ref<'login' | 'register'>('login')
const form = reactive({ username: 'captain', password: 'captain123', nickname: '' })
const errs = reactive({ username: '', password: '', nickname: '' })
const auth = useAuth()
const router = useRouter()

function validate() {
  errs.username = form.username.trim().length < 3 ? '用户名至少 3 位' : ''
  errs.password = form.password.length < 6 ? '密码至少 6 位' : ''
  errs.nickname = tab.value === 'register' && !form.nickname.trim() ? '请填写昵称' : ''
  return !errs.username && !errs.password && !errs.nickname
}

async function submit() {
  if (!validate()) {
    toast('请先修正表单', 'err')
    return
  }
  try {
    if (tab.value === 'login') await auth.login(form.username, form.password)
    else await auth.register(form.username, form.password, form.nickname)
    router.push('/')
  } catch (e: any) {
    toast(e.message || '失败', 'err')
  }
}
</script>

<template>
  <div class="flex min-h-full items-center justify-center px-4">
    <div class="w-full max-w-md rounded-3xl border border-paper/10 bg-dusk/90 p-8 shadow-2xl">
      <p class="text-xs uppercase tracking-[0.3em] text-lantern">Mini 旅行路书</p>
      <h1 class="mt-2 font-display text-4xl">夜徒出发前</h1>
      <p class="mt-2 text-sm text-paper/60">把路书写进同一张地图，再把脚印实时叠上去。</p>
      <div class="mt-6 flex gap-2 text-sm">
        <button :class="tab==='login' ? 'bg-moss text-paper' : 'bg-ink text-paper/70'" class="rounded-full px-4 py-1" @click="tab='login'">登录</button>
        <button :class="tab==='register' ? 'bg-moss text-paper' : 'bg-ink text-paper/70'" class="rounded-full px-4 py-1" @click="tab='register'">注册</button>
      </div>
      <form class="mt-6 space-y-4" @submit.prevent="submit">
        <label class="block text-sm">用户名 *
          <input v-model="form.username" class="mt-1 w-full rounded-xl border border-paper/10 bg-ink px-3 py-2" />
          <div v-if="errs.username" class="field-err">{{ errs.username }}</div>
        </label>
        <label class="block text-sm">密码 *
          <input v-model="form.password" type="password" class="mt-1 w-full rounded-xl border border-paper/10 bg-ink px-3 py-2" />
          <div v-if="errs.password" class="field-err">{{ errs.password }}</div>
        </label>
        <label v-if="tab==='register'" class="block text-sm">昵称 *
          <input v-model="form.nickname" class="mt-1 w-full rounded-xl border border-paper/10 bg-ink px-3 py-2" />
          <div v-if="errs.nickname" class="field-err">{{ errs.nickname }}</div>
        </label>
        <button type="submit" class="w-full rounded-2xl bg-lantern py-2.5 font-medium text-ink">
          {{ tab === 'login' ? '进入营地' : '创建身份' }}
        </button>
      </form>
      <p class="mt-4 text-xs text-paper/45">演示账号 captain / captain123 · 邀请码 HTK8M2</p>
    </div>
  </div>
</template>
