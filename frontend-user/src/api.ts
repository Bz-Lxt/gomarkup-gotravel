export type Envelope<T> = { ok: boolean; data?: T; code?: string; error?: { message: string; detail?: unknown } }

const TOKEN_KEY = 'gt_token'

export function getToken() { return localStorage.getItem(TOKEN_KEY) || '' }
export function setToken(t: string) { localStorage.setItem(TOKEN_KEY, t) }
export function clearToken() { localStorage.removeItem(TOKEN_KEY) }

export function fmtTime(isoOrDate: string | Date) {
  const d = typeof isoOrDate === 'string' ? new Date(isoOrDate) : isoOrDate
  if (Number.isNaN(d.getTime())) return String(isoOrDate)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (!(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  const tok = getToken()
  if (tok) headers.set('Authorization', `Bearer ${tok}`)
  const res = await fetch(path, { ...init, headers })
  const body = (await res.json()) as Envelope<T>
  if (!res.ok || !body.ok) {
    const err = new Error(body.error?.message || '请求失败') as Error & { code?: string; detail?: unknown }
    err.code = body.code
    err.detail = body.error?.detail
    throw err
  }
  return body.data as T
}

export const AuthAPI = {
  login: (username: string, password: string) =>
    api<{ user: User; token: string }>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
  register: (username: string, password: string, nickname: string) =>
    api<{ user: User; token: string }>('/api/v1/auth/register', { method: 'POST', body: JSON.stringify({ username, password, nickname }) }),
  me: () => api<User>('/api/v1/auth/me'),
}

export type User = { id: number; username: string; nickname: string; avatar_color: string }
export type Team = { id: number; name: string; leader_id: number; invite_code: string; role?: string }
export type Member = { user_id: number; username: string; nickname: string; avatar_color: string; role: string }
export type Trip = { id: number; team_id: number; title: string; description: string; total_distance_m: number; status: string }
export type Waypoint = { id: number; trip_id: number; seq: number; name: string; kind: string; lat: number; lng: number; note: string; planned_stay_min: number }
export type Distance = { total_meters: number; segments: { meters: number }[]; eta_minutes: number; provider: string }
export type Session = { id: number; trip_id: number; status: string; started_at: string }
export type Photo = { id: number; lat: number; lng: number; url: string; thumb_url: string; caption: string; nickname: string; created_at: string }

export const TeamAPI = {
  list: () => api<Team[]>('/api/v1/teams'),
  create: (name: string) => api<Team>('/api/v1/teams', { method: 'POST', body: JSON.stringify({ name }) }),
  join: (code: string) => api<Team>('/api/v1/teams/join', { method: 'POST', body: JSON.stringify({ code }) }),
  get: (id: number) => api<{ team: Team; members: Member[] }>(`/api/v1/teams/${id}`),
}

export const TripAPI = {
  list: (teamId: number) => api<Trip[]>(`/api/v1/teams/${teamId}/trips`),
  create: (teamId: number, title: string, description: string) =>
    api<Trip>(`/api/v1/teams/${teamId}/trips`, { method: 'POST', body: JSON.stringify({ title, description }) }),
  get: (id: number) => api<{ trip: Trip; waypoints: Waypoint[]; deps: unknown[] }>(`/api/v1/trips/${id}`),
  addWP: (id: number, body: Partial<Waypoint>) =>
    api<{ waypoint: Waypoint; distance: Distance }>(`/api/v1/trips/${id}/waypoints`, { method: 'POST', body: JSON.stringify(body) }),
  reorder: (id: number, ids: number[]) =>
    api<Distance>(`/api/v1/trips/${id}/waypoints/reorder`, { method: 'PUT', body: JSON.stringify({ ids }) }),
  delWP: (id: number) => api<Distance>(`/api/v1/waypoints/${id}`, { method: 'DELETE' }),
  start: (tripId: number) => api<Session>(`/api/v1/trips/${tripId}/sessions`, { method: 'POST' }),
  photos: (tripId: number) => api<Photo[]>(`/api/v1/trips/${tripId}/photos`),
  upload: (tripId: number, fd: FormData) => api<Photo>(`/api/v1/trips/${tripId}/photos`, { method: 'POST', body: fd }),
}

export const SessAPI = {
  get: (id: number) => api<{ session: Session; trip: Trip }>(`/api/v1/sessions/${id}`),
  rally: (id: number, lat: number, lng: number, message: string) =>
    api<unknown>(`/api/v1/sessions/${id}/rally`, { method: 'POST', body: JSON.stringify({ lat, lng, message }) }),
  ack: (id: number) => api<unknown>(`/api/v1/alerts/${id}/ack`, { method: 'POST' }),
}

export const SimAPI = {
  start: (session_id: number, count: number, laggard: boolean) =>
    api<unknown>('/api/v1/sim/start', { method: 'POST', body: JSON.stringify({ session_id, count, laggard }) }),
  stop: () => api<unknown>('/api/v1/sim/stop', { method: 'POST' }),
}
