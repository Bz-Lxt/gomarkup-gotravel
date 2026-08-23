import L from 'leaflet'

export function offlineGrid() {
  return L.gridLayer({
    tileSize: 256,
    createTile(coords) {
      const c = document.createElement('canvas')
      c.width = 256
      c.height = 256
      const g = c.getContext('2d')!
      g.fillStyle = '#15241c'
      g.fillRect(0, 0, 256, 256)
      g.strokeStyle = '#2a4034'
      g.strokeRect(0.5, 0.5, 255, 255)
      g.fillStyle = '#8aa392'
      g.font = '11px Outfit, sans-serif'
      g.fillText(`${coords.z}/${coords.x}/${coords.y}`, 8, 20)
      return c
    },
  })
}

export function addBase(map: L.Map) {
  const osm = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; OSM',
  })
  let failed = 0
  osm.on('tileerror', () => {
    failed++
    if (failed >= 3) {
      map.removeLayer(osm)
      offlineGrid().addTo(map)
    }
  })
  osm.addTo(map)
}

export const kindColor: Record<string, string> = {
  STOP: '#2F6F4E',
  CHECKIN: '#E8A04A',
  LODGING: '#C46A2B',
}

export function kindLabel(k: string) {
  return ({ STOP: '经停', CHECKIN: '打卡', LODGING: '住宿' } as Record<string, string>)[k] || k
}

export function km(m: number) {
  if (m < 1000) return `${m.toFixed(0)} m`
  return `${(m / 1000).toFixed(2)} km`
}
