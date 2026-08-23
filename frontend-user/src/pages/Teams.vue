<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import Shell from '../components/Shell.vue'
import Modal from '../components/Modal.vue'
import { TeamAPI, TripAPI, type Team, type Trip } from '../api'
import { toast } from '../toast'
import { km } from '../maputil'

const teams = ref<Team[]>([])
const active = ref<number | null>(null)
const trips = ref<Trip[]>([])
const members = ref<string>('')
const showCreate = ref(false)
const showJoin = ref(false)
const showTrip = ref(false)
const form = reactive({ name: '', code: '', title: '', desc: '' })
const ferr = reactive({ name: '', code: '', title: '' })
const router = useRouter()

async function load() {
  teams.value = await TeamAPI.list()
  if (!active.value && teams.value.length) active.value = teams.value[0].id
  if (active.value) await open(active.value)
}

async function open(id: number) {
  active.value = id
  const d = await TeamAPI.get(id)
  members.value = d.members.map((m) => `${m.nickname}(${m.role === 'leader' ? '队长' : '队员'})`).join(' · ')
  trips.value = await TripAPI.list(id)
}

async function createTeam() {
  ferr.name = form.name.trim() ? '' : '队伍名必填'
  if (ferr.name) { toast('请先修正表单', 'err'); return }
  try {
    const t = await TeamAPI.create(form.name)
    showCreate.value = false
    form.name = ''
    await load()
    await open(t.id)
    toast('队伍已立旗')
  } catch (e: any) { toast(e.message, 'err') }
}

async function join() {
  ferr.code = form.code.trim().length === 6 ? '' : '邀请码须 6 位'
  if (ferr.code) { toast('请先修正表单', 'err'); return }
  try {
    const t = await TeamAPI.join(form.code)
    showJoin.value = false
    form.code = ''
    await load()
    await open(t.id)
    toast('已加入队伍')
  } catch (e: any) { toast(e.message, 'err') }
}

async function createTrip() {
  if (!active.value) return
  ferr.title = form.title.trim() ? '' : '路书标题必填'
  if (ferr.title) { toast('请先修正表单', 'err'); return }
  try {
    const t = await TripAPI.create(active.value, form.title, form.desc)
    showTrip.value = false
    form.title = ''
    form.desc = ''
    router.push(`/trips/${t.id}`)
  } catch (e: any) { toast(e.message, 'err') }
}

onMounted(() => load().catch((e) => toast(e.message, 'err')))
</script>

<template>
  <Shell title="营地" sub="组队 · 路书 · 出发">
    <div class="grid w-full gap-0 lg:grid-cols-[320px_1fr]">
      <aside class="border-r border-paper/10 p-5">
        <div class="mb-4 flex gap-2">
          <button class="rounded-full bg-moss px-3 py-1 text-sm" @click="showCreate=true">建队</button>
          <button class="rounded-full border border-paper/20 px-3 py-1 text-sm" @click="showJoin=true">加入</button>
        </div>
        <button
          v-for="t in teams" :key="t.id"
          class="mb-2 block w-full rounded-2xl border px-4 py-3 text-left"
          :class="active===t.id ? 'border-lantern bg-lantern/10' : 'border-paper/10 bg-dusk/40'"
          @click="open(t.id)"
        >
          <div class="font-display text-lg">{{ t.name }}</div>
          <div class="text-xs text-paper/50">邀请码 {{ t.invite_code }}</div>
        </button>
        <p v-if="!teams.length" class="text-sm text-paper/50">还没有队伍。先立一面旗。</p>
      </aside>
      <section class="p-6">
        <div class="text-sm text-paper/60">{{ members }}</div>
        <div class="mt-4 flex items-center justify-between">
          <h2 class="font-display text-3xl">路书</h2>
          <button class="rounded-full bg-lantern px-4 py-1.5 text-ink" @click="showTrip=true">新路书</button>
        </div>
        <div class="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <article v-for="tr in trips" :key="tr.id" class="rounded-3xl border border-paper/10 bg-dusk/50 p-5">
            <h3 class="font-display text-2xl">{{ tr.title }}</h3>
            <p class="mt-1 text-sm text-paper/55">{{ tr.description || '尚未写下备注' }}</p>
            <p class="mt-3 text-lantern">{{ km(tr.total_distance_m) }}</p>
            <div class="mt-4 flex flex-wrap gap-2 text-sm">
              <button class="rounded-full bg-moss px-3 py-1" @click="router.push(`/trips/${tr.id}`)">编辑路书</button>
              <button class="rounded-full border border-paper/20 px-3 py-1" @click="router.push(`/journal/${tr.id}`)">照片墙</button>
            </div>
          </article>
        </div>
      </section>
    </div>
    <Modal v-if="showCreate" title="建立队伍" @close="showCreate=false">
      <label class="block text-sm">队伍名 *
        <input v-model="form.name" class="mt-1 w-full rounded-xl bg-ink px-3 py-2" />
        <div v-if="ferr.name" class="field-err">{{ ferr.name }}</div>
      </label>
      <button class="mt-4 w-full rounded-xl bg-moss py-2" @click="createTeam">确认</button>
    </Modal>
    <Modal v-if="showJoin" title="凭码加入" @close="showJoin=false">
      <label class="block text-sm">邀请码 *
        <input v-model="form.code" class="mt-1 w-full rounded-xl bg-ink px-3 py-2 uppercase" />
        <div v-if="ferr.code" class="field-err">{{ ferr.code }}</div>
      </label>
      <button class="mt-4 w-full rounded-xl bg-moss py-2" @click="join">加入</button>
    </Modal>
    <Modal v-if="showTrip" title="新路书" @close="showTrip=false">
      <label class="block text-sm">标题 *
        <input v-model="form.title" class="mt-1 w-full rounded-xl bg-ink px-3 py-2" />
        <div v-if="ferr.title" class="field-err">{{ ferr.title }}</div>
      </label>
      <label class="mt-3 block text-sm">备注
        <textarea v-model="form.desc" class="mt-1 w-full rounded-xl bg-ink px-3 py-2" />
      </label>
      <button class="mt-4 w-full rounded-xl bg-lantern py-2 text-ink" @click="createTrip">开始编排</button>
    </Modal>
  </Shell>
</template>
