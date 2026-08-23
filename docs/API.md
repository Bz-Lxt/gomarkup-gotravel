# GoTravel API

Base URL：`/api/v1`  
时区：GMT+8，展示格式 `yyyy-MM-dd HH:mm:ss`  
统一包络：`{"ok":true,"data":...}` / `{"ok":false,"code":"VALIDATION","error":{"message":"..."}}`

## 错误码

| code | HTTP | 含义 |
|---|---|---|
| BAD_REQUEST | 400 | JSON 非法 |
| VALIDATION | 400 | 字段校验失败 |
| UNAUTHORIZED | 401 | 未登录/Token 无效 |
| FORBIDDEN | 403 | 非队员或非队长 |
| NOT_FOUND | 404 | 资源不存在 |
| CONFLICT | 409 | 用户名占用等 |
| CYCLE_DETECTED | 409 | 依赖成环，`detail` 为节点链 |
| RATE_LIMITED | 429 | WS 上行超限 |
| PAYLOAD_TOO_LARGE | 400 | 照片 > 10MB |
| INTERNAL | 500 | 未处理错误 |

鉴权：`Authorization: Bearer <jwt>`；WebSocket 可用 `?token=`。

## REST

### POST /auth/register
```json
{"username":"alice","password":"alice123","nickname":"爱丽丝"}
```
响应 201：`{"user":{"id":1,"username":"alice","nickname":"爱丽丝","avatar_color":"#2F6F4E"},"token":"..."}`

### POST /auth/login
```json
{"username":"captain","password":"captain123"}
```

### GET /auth/me
返回当前用户。

### POST /teams  / GET /teams / POST /teams/join / GET /teams/{id}
创建：`{"name":"西湖夜徒"}`  
加入：`{"code":"HTK8M2"}`  
详情：`{"team":{...},"members":[...]}`

### POST /teams/{id}/trips  GET /teams/{id}/trips
```json
{"title":"西线夜徒","description":"..."}
```

### GET /trips/{id}  PATCH /trips/{id}  DELETE /trips/{id}
详情含 `waypoints` 与 `deps`。

### POST /trips/{id}/waypoints
```json
{"name":"断桥","kind":"CHECKIN","lat":30.2578,"lng":120.1512,"note":"集合"}
```
kind：`STOP|CHECKIN|LODGING`

### PUT /trips/{id}/waypoints/reorder
```json
{"ids":[3,1,2]}
```

### POST /trips/{id}/waypoints/import
Body 为 GeoJSON FeatureCollection，坐标 `[lng,lat]`。

### POST /trips/{id}/deps
```json
{"from_waypoint_id":1,"to_waypoint_id":3}
```
成环时 `code=CYCLE_DETECTED`，`detail` 为环路 id 列表。

### GET /trips/{id}/topo  GET /trips/{id}/distance
### PATCH|DELETE /waypoints/{id}

### POST /trips/{id}/sessions  GET /sessions/{id}  POST /sessions/{id}/end
### GET /sessions/{id}/positions
### POST /sessions/{id}/rally
```json
{"lat":30.25,"lng":120.15,"message":"原地集合"}
```
### GET /sessions/{id}/alerts  POST /alerts/{id}/ack

### POST /trips/{id}/photos  (multipart)
字段：`file, lat, lng, caption, session_id?`  
### GET /trips/{id}/photos

### POST /sim/start
```json
{"session_id":1,"count":8,"laggard":true}
```
### POST /sim/stop  GET /sim/status
### GET /health  GET /metrics

## WebSocket `/ws?token=&session_id=`

上行：
```json
{"type":"pos","lat":30.25,"lng":120.15,"speed":1.2,"accuracy":8,"heading":90,"ts":1724400000000}
{"type":"ping"}
{"type":"ack","alert_id":1}
```

下行：
- `welcome` `{session_id,self_id}`
- `batch_pos` `{members:[{user_id,nickname,avatar_color,lat,lng,net,ts}],tick}`
- `rally` 紧急集合（优先通道，不丢弃）
- `laggard` 掉队预警（优先通道）
- `presence` 离线
- `error` `{code:"RATE_LIMITED"}`

net：`online|weak|offline`（3s / 15s）。
