<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import L from 'leaflet'
import Shell from '../components/Shell.vue'
import { TripAPI, fmtTime, type Photo, type Trip } from '../api'
import { addBase } from '../maputil'
import { toast } from '../toast'

const route = useRoute()
const tripId = Number(route.params.id)
const trip = ref<Trip | null>(null)
const photos = ref<Photo[]>([])
const caption = ref('')
const pick = ref<{ lat: number; lng: number } | null>(null)
const file = ref<File | null>(null)
const mapEl = ref<HTMLElement | null>(null)
const focus = ref<Photo | null>(null)
let map: L.Map | null = null

async function load() {
  const d = await TripAPI.get(tripId)
  trip.value = d.trip
  photos.value = await TripAPI.photos(tripId)
  await nextTick()
  if (!map && mapEl.value) {
    map = L.map(mapEl.value).setView([30.25, 120.14], 13)
    addBase(map)
    map.on('click', (e: L.LeafletMouseEvent) => { pick.value = { lat: e.latlng.lat, lng: e.latlng.lng } })
  }
  photos.value.forEach((p) => {
    L.circleMarker([p.lat, p.lng], { radius: 8, color: '#E8A04A', fillColor: '#C46A2B', fillOpacity: 0.9 })
      .bindTooltip(p.caption || p.nickname)
      .addTo(map!)
  })
}

function onFile(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  file.value = f || null
}

async function upload() {
  if (!file.value) { toast('请选择照片', 'err'); return }
  if (!pick.value) { toast('请先在地图点选坐标', 'err'); return }
  const fd = new FormData()
  fd.append('file', file.value)
  fd.append('lat', String(pick.value.lat))
  fd.append('lng', String(pick.value.lng))
  fd.append('caption', caption.value)
  try {
    await TripAPI.upload(tripId, fd)
    caption.value = ''
    file.value = null
    toast('足迹已钉上墙')
    await load()
  } catch (e: any) { toast(e.message, 'err') }
}

function locate(p: Photo) {
  focus.value = p
  map?.flyTo([p.lat, p.lng], 16)
}

onMounted(() => load().catch((e) => toast(e.message, 'err')))
</script>

<template>
  <Shell :title="trip?.title || '旅行手账'" sub="先点地图，再上传照片">
    <div class="grid min-h-[calc(100vh-72px)] w-full lg:grid-cols-[1fr_420px]">
      <div ref="mapEl" class="map-ink min-h-[360px] w-full"></div>
      <section class="border-l border-paper/10 p-5">
        <div class="rounded-2xl bg-dusk/60 p-4 text-sm">
          <div>选中坐标：{{ pick ? `${pick.lat.toFixed(5)}, ${pick.lng.toFixed(5)}` : '尚未点选' }}</div>
          <input type="file" accept="image/*" class="mt-3 w-full text-xs" @change="onFile" />
          <input v-model="caption" class="mt-2 w-full rounded-xl bg-ink px-3 py-2" placeholder="一句手账" />
          <button class="mt-3 w-full rounded-xl bg-lantern py-2 text-ink" @click="upload">钉到足迹墙</button>
        </div>
        <div class="perspective mt-8 h-[480px]">
          <article
            v-for="(p, i) in photos" :key="p.id"
            class="absolute w-56 cursor-pointer rounded-2xl border border-paper/20 bg-paper p-2 text-ink shadow-2xl transition hover:-translate-y-3 hover:rotate-0"
            :style="{ left: `${(i % 3) * 28 + 8}px`, top: `${Math.floor(i / 3) * 36 + (i % 2) * 12}px`, transform: `rotate(${(i % 5 - 2) * 4}deg) translateZ(${i * 8}px)` }"
            @click="locate(p)"
          >
            <img :src="p.thumb_url || p.url" class="h-36 w-full rounded-xl object-cover" alt="" />
            <div class="mt-2 font-display text-sm">{{ p.caption || '未命名足迹' }}</div>
            <div class="text-[11px] text-ink/60">{{ p.nickname }} · {{ fmtTime(p.created_at) }}</div>
          </article>
          <p v-if="!photos.length" class="text-sm text-paper/50">墙还是空的。去地图上按下一个坐标。</p>
        </div>
      </section>
    </div>
  </Shell>
</template>

<style scoped>
.perspective { perspective: 900px; position: relative; }
</style>
