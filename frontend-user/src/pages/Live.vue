<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import L from 'leaflet'
import Shell from '../components/Shell.vue'
import Modal from '../components/Modal.vue'
import { SessAPI, SimAPI, getToken, type Trip } from '../api'
import { addBase } from '../maputil'
import { toast } from '../toast'
import { useAuth } from '../store'

type Member = {
  user_id: number; nickname: string; avatar_color: string; lat: number; lng: number; net: string; ts: number
}

const route = useRoute()
const sid = Number(route.params.id)
const auth = useAuth()
const trip = ref<Trip | null>(null)
const members = ref<Member[]>([])
const rallyOpen = ref(false)
const rallyMsg = ref('紧急集合，立刻到队长位置会合')
const alertBox = ref<{ id: number; title: string; body: string } | null>(null)
const mapEl = ref<HTMLElement | null>(null)
let map: L.Map | null = null
let ws: WebSocket | null = null
const markers = new Map<number, { m: L.Marker; from: L.LatLng; to: L.LatLng; t0: number }>()
let raf = 0

function icon(m: Member) {
  const ring = m.net === 'online' ? '#2F6F4E' : m.net === 'weak' ? '#E8A04A' : '#6B7280'
  return L.divIcon({
    className: '',
    html: `<div style="width:36px;height:36px;border-radius:50%;border:3px solid ${ring};background:${m.avatar_color};display:flex;align-items:center;justify-content:center;color:#F3E8D4;font-size:12px;box-shadow:0 8px 16px rgba(0,0,0,.35)">${(m.nickname || '?').slice(0, 1)}</div>`,
    iconSize: [36, 36], iconAnchor: [18, 18],
  })
}

function upsert(list: Member[]) {
  const now = performance.now()
  list.forEach((p) => {
    const i = members.value.findIndex((x) => x.user_id === p.user_id)
    if (i >= 0) members.value[i] = p
    else members.value.push(p)
    const ll = L.latLng(p.lat, p.lng)
    const rec = markers.get(p.user_id)
    if (!rec) {
      const mk = L.marker(ll, { icon: icon(p) }).addTo(map!)
      markers.set(p.user_id, { m: mk, from: ll, to: ll, t0: now })
    } else {
      rec.from = rec.m.getLatLng()
      rec.to = ll
      rec.t0 = now
      rec.m.setIcon(icon(p))
    }
  })
}

function tick() {
  const now = performance.now()
  markers.forEach((rec) => {
    const u = Math.min(1, (now - rec.t0) / 220)
    const lat = rec.from.lat + (rec.to.lat - rec.from.lat) * u
    const lng = rec.from.lng + (rec.to.lng - rec.from.lng) * u
    rec.m.setLatLng([lat, lng])
  })
  raf = requestAnimationFrame(tick)
}

function connect() {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  ws = new WebSocket(`${proto}://${location.host}/ws?token=${getToken()}&session_id=${sid}`)
  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data)
    if (msg.type === 'batch_pos') upsert(msg.members || [])
    if (msg.type === 'rally') {
      alertBox.value = { id: msg.alert_id, title: '紧急集合', body: msg.message }
      try { new AudioContext() } catch { /* ignore */ }
    }
    if (msg.type === 'laggard') {
      toast(`${msg.nickname || '队员'} 掉队 ${Math.round(msg.distance_m)} 米`, 'warn')
    }
    if (msg.type === 'error') toast(msg.message, 'err')
  }
  ws.onclose = () => { setTimeout(() => { if (!ws || ws.readyState === 3) connect() }, 1500) }
}

async function rally() {
  const self = members.value.find((m) => m.user_id === auth.user?.id) || members.value[0]
  if (!self) { toast('还没有位置', 'err'); return }
  try {
    await SessAPI.rally(sid, self.lat, self.lng, rallyMsg.value)
    rallyOpen.value = false
    toast('集合令已发出')
  } catch (e: any) { toast(e.message, 'err') }
}

async function ack() {
  if (!alertBox.value) return
  try {
    await SessAPI.ack(alertBox.value.id)
    ws?.send(JSON.stringify({ type: 'ack', alert_id: alertBox.value.id }))
  } catch { /* ignore */ }
  alertBox.value = null
}

onMounted(async () => {
  try {
    const d = await SessAPI.get(sid)
    trip.value = d.trip
    await nextTick()
    map = L.map(mapEl.value!, { zoomControl: true }).setView([30.25, 120.14], 13)
    addBase(map)
    raf = requestAnimationFrame(tick)
    connect()
  } catch (e: any) { toast(e.message, 'err') }
})
onUnmounted(() => {
  cancelAnimationFrame(raf)
  ws?.close()
  map?.remove()
})
</script>

<template>
  <Shell :title="trip?.title || '实时足迹'" sub="头像描边：绿在线 / 琥珀弱网 / 灰离线">
    <div class="relative h-[calc(100vh-72px)] w-full">
      <div ref="mapEl" class="map-ink h-full w-full"></div>
      <aside class="absolute left-4 top-4 z-[500] w-72 rounded-2xl border border-paper/10 bg-ink/85 p-4 backdrop-blur">
        <div class="font-display text-xl">队伍雷达</div>
        <ul class="mt-3 space-y-2 text-sm">
          <li v-for="m in members" :key="m.user_id" class="flex items-center justify-between">
            <span>{{ m.nickname }}</span>
            <span :class="m.net==='online'?'text-moss':m.net==='weak'?'text-lantern':'text-paper/40'">{{ m.net }}</span>
          </li>
        </ul>
        <button class="mt-4 w-full rounded-xl bg-clay py-2 text-sm" @click="rallyOpen=true">紧急集合</button>
        <button class="mt-2 w-full rounded-xl border border-paper/20 py-2 text-sm" @click="SimAPI.start(sid, 8, true).then(()=>toast('模拟队员已上线')).catch((e:any)=>toast(e.message,'err'))">启动 8 人模拟（含掉队）</button>
        <button class="mt-2 w-full rounded-xl border border-paper/10 py-1 text-xs text-paper/50" @click="SimAPI.stop()">停止模拟</button>
      </aside>
    </div>
    <Modal v-if="rallyOpen" title="向全员发送集合" @close="rallyOpen=false">
      <textarea v-model="rallyMsg" class="w-full rounded-xl bg-ink p-2" />
      <button class="mt-3 w-full rounded-xl bg-clay py-2" @click="rally">发送（不可丢弃通道）</button>
    </Modal>
    <Modal v-if="alertBox" :title="alertBox.title" @close="ack">
      <p>{{ alertBox.body }}</p>
      <button class="mt-4 w-full rounded-xl bg-lantern py-2 text-ink" @click="ack">已知晓</button>
    </Modal>
  </Shell>
</template>
