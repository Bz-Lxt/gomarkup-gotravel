<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import L from 'leaflet'
import Shell from '../components/Shell.vue'
import { TripAPI, type Distance, type Trip, type Waypoint } from '../api'
import { addBase, kindColor, kindLabel, km } from '../maputil'
import { toast } from '../toast'

const route = useRoute()
const router = useRouter()
const tripId = Number(route.params.id)
const trip = ref<Trip | null>(null)
const wps = ref<Waypoint[]>([])
const dist = ref<Distance | null>(null)
const kind = ref('CHECKIN')
const name = ref('')
const mapEl = ref<HTMLElement | null>(null)
let map: L.Map | null = null
let line: L.Polyline | null = null
const markers: L.CircleMarker[] = []
const dragFrom = ref<number | null>(null)

async function load() {
  const d = await TripAPI.get(tripId)
  trip.value = d.trip
  wps.value = d.waypoints || []
  dist.value = { total_meters: d.trip.total_distance_m, segments: [], eta_minutes: d.trip.total_distance_m / 1000 / 4.5 * 60, provider: 'haversine' }
  await nextTick()
  draw()
}

function ensureMap() {
  if (map || !mapEl.value) return
  map = L.map(mapEl.value, { zoomControl: true }).setView([30.25, 120.14], 13)
  addBase(map)
  map.on('click', async (e: L.LeafletMouseEvent) => {
    try {
      const r = await TripAPI.addWP(tripId, {
        name: name.value || `点 ${wps.value.length + 1}`,
        kind: kind.value,
        lat: e.latlng.lat,
        lng: e.latlng.lng,
      })
      name.value = ''
      dist.value = r.distance
      await load()
    } catch (err: any) { toast(err.message, 'err') }
  })
}

function draw() {
  ensureMap()
  if (!map) return
  markers.splice(0).forEach((m) => m.remove())
  line?.remove()
  const latlngs = wps.value.map((w) => [w.lat, w.lng] as [number, number])
  if (latlngs.length) {
    line = L.polyline(latlngs, { color: '#E8A04A', weight: 4, opacity: 0.85 }).addTo(map)
    wps.value.forEach((w, i) => {
      const m = L.circleMarker([w.lat, w.lng], {
        radius: 9, color: '#0E1612', weight: 2, fillColor: kindColor[w.kind] || '#2F6F4E', fillOpacity: 1,
      }).bindTooltip(`${i + 1}. ${w.name} · ${kindLabel(w.kind)}`).addTo(map!)
      markers.push(m)
    })
    map.fitBounds(L.latLngBounds(latlngs).pad(0.2))
  }
}

async function remove(id: number) {
  try {
    dist.value = await TripAPI.delWP(id)
    await load()
  } catch (e: any) { toast(e.message, 'err') }
}

function onDragStart(i: number) { dragFrom.value = i }
async function onDrop(i: number) {
  if (dragFrom.value == null || dragFrom.value === i) return
  const arr = [...wps.value]
  const [moved] = arr.splice(dragFrom.value, 1)
  arr.splice(i, 0, moved)
  wps.value = arr
  dragFrom.value = null
  try {
    dist.value = await TripAPI.reorder(tripId, arr.map((w) => w.id))
    draw()
  } catch (e: any) { toast(e.message, 'err') }
}

async function startLive() {
  try {
    const s = await TripAPI.start(tripId)
    router.push(`/live/${s.id}`)
  } catch (e: any) { toast(e.message, 'err') }
}

onMounted(() => load().catch((e) => toast(e.message, 'err')))
onUnmounted(() => { map?.remove(); map = null })
</script>

<template>
  <Shell :title="trip?.title || '路书'" sub="点击地图落点 · 拖拽左侧列表改顺序">
    <div class="grid h-[calc(100vh-72px)] w-full lg:grid-cols-[360px_1fr]">
      <aside class="overflow-auto border-r border-paper/10 p-4">
        <div class="rounded-2xl bg-dusk/70 p-4">
          <div class="text-3xl font-display text-lantern">{{ km(dist?.total_meters || 0) }}</div>
          <div class="text-xs text-paper/50">预估徒步 {{ (dist?.eta_minutes || 0).toFixed(0) }} 分钟 · {{ dist?.provider }}</div>
          <div class="mt-3 flex gap-2">
            <button class="rounded-full bg-clay px-3 py-1 text-sm" @click="startLive">开始出行</button>
            <button class="rounded-full border border-paper/20 px-3 py-1 text-sm" @click="router.push(`/journal/${tripId}`)">照片墙</button>
          </div>
        </div>
        <label class="mt-4 block text-sm">点名
          <input v-model="name" class="mt-1 w-full rounded-xl bg-dusk px-3 py-2" placeholder="可选，默认自动编号" />
        </label>
        <label class="mt-2 block text-sm">类型
          <select v-model="kind" class="mt-1 w-full rounded-xl bg-dusk px-3 py-2">
            <option value="STOP">经停点</option>
            <option value="CHECKIN">打卡点</option>
            <option value="LODGING">住宿点</option>
          </select>
        </label>
        <ul class="mt-4 space-y-2">
          <li
            v-for="(w, i) in wps" :key="w.id"
            draggable="true"
            class="flex cursor-grab items-center justify-between rounded-2xl border border-paper/10 bg-dusk/50 px-3 py-2"
            @dragstart="onDragStart(i)"
            @dragover.prevent
            @drop="onDrop(i)"
          >
            <div>
              <div class="text-sm">{{ i + 1 }}. {{ w.name }}</div>
              <div class="text-xs" :style="{ color: kindColor[w.kind] }">{{ kindLabel(w.kind) }}</div>
            </div>
            <button class="text-xs text-clay" @click="remove(w.id)">删除</button>
          </li>
        </ul>
      </aside>
      <div ref="mapEl" class="map-ink h-full min-h-[420px] w-full"></div>
    </div>
  </Shell>
</template>
