# GoTravel · 实施路线图

| 项目 | 内容 |
|---|---|
| 文档版本 | v1.0 |
| 对应需求 | `docs/Requirements.md` v1.0 |
| 制定时间 | 2026-08-23 17:30 (GMT+8) |
| 制定角色 | Chief Architect |
| 规模档位 | 10k–40k LoC → **分期强制** |

---

## 0. Phase Order Decision（v13 §4 Phase 1.5）

**决策：Logic-First（交换 Phase 2 与 Phase 3）。**

理由：地图路书编辑器、实时轨迹画布与照片墙的组件树直接由 Waypoint / DAG 边 / 位置帧协议决定，先画 UI 会在协议定型后整页返工。

执行顺序：Phase 1 架构 → Phase 3 后端与协议 → Phase 2 前端对接真实 API → Phase 4 QA → Phase 5 审核。

---

## 1. 阶段边界（强制）

### MVP（本轮 `/auto` 交付范围 · P0）

- 账号 JWT、队伍邀请码、队长/队员角色
- 路书编辑器：点击打点、三类点（STOP/CHECKIN/LODGING）、拖拽排序、Haversine 总里程
- 出行实例 + WebSocket 实时位置大屏（平滑插值、在线/弱网/离线）
- 紧急集合（不可丢弃通道 + 已读回执）
- 掉队预警（空间索引 + 60s 去抖）
- 足迹照片墙（坐标上传 + 缩略图 + 立体层叠）
- **手写** DAG 拓扑排序 / 环检测（服务端，≥50 节点）
- **手写** R-Tree + Redis Geo 双实现，`GEO_INDEX_BACKEND` 切换
- **手写** Channel 扇出：有界缓冲、Drop-Oldest、200ms 合并、令牌桶、慢消费者踢出、优先通道
- GPSSimulator Mock Provider（沿折线插值，含掉队者）
- Docker Compose 一键启动，随机端口 27181–27184
- 单元测试覆盖 DAG / R-Tree / Haversine / 扇出；Playwright 关键路径

### V1（本轮不实现，仅划界）

- 队长转让 / 移除队员
- 依赖树可视化编辑（前端 DAG）
- GeoJSON / GPX 导出
- 历史轨迹回放、队伍进度条
- 照片点聚合（Cluster）

### V2（本轮不实现，仅划界）

- 路书模板库
- 按天旅行日报
- 队员间一对一文字消息

---

## 2. 目录结构

```
GoTravel/
├── backend/                 # Go 服务
├── frontend-user/           # Vue 3 用户端（唯一交付 UI）
├── frontend-admin/          # 占位：需求未含运营后台
├── frontend-mp/             # 占位：需求未含小程序
├── tests/                   # Playwright + API smoke
├── docker-compose.yml
└── docs/
```

范围漂移约束：`frontend-admin` / `frontend-mp` 仅 README 占位，不实现业务。

---

## 3. 技术栈冻结

| 层 | 选型 |
|---|---|
| 后端 | Go 1.25 + stdlib ServeMux + pgx + go-redis + gorilla/websocket + bcrypt + JWT |
| 前端 | Vue 3 + Vite + TypeScript + Tailwind + Pinia + Vue Router + Leaflet 1.9 + OSM |
| 数据 | PostgreSQL 16 + Redis 7 |
| 地图降级 | 前端离线网格底图（无 Key） |
| 里程 | 默认 Haversine；`ROUTE_PROVIDER=haversine\|osrm` |
| 空间索引 | `GEO_INDEX_BACKEND=rtree\|redis`（Redis 不可用自动降级 rtree） |
| GPS | `GPS_PROVIDER=live\|sim` |
| 时区 | Asia/Shanghai / GMT+8 |

---

## 4. 任务拆分（MVP）

| ID | 任务 | 阶段 | 状态 |
|---|---|---|---|
| A-01 | Git / gitignore / compose 随机端口骨架 | 1 | [x] |
| L-01 | 配置 / 日志 / GMT+8 / 错误码 / 迁移锁 | 3 | [x] |
| L-02 | 用户/队伍/路书/路径点 CRUD | 3 | [x] |
| L-03 | routegraph DAG + 拓扑 + 环检测 | 3 | [x] |
| L-04 | geo Index + R-Tree + RedisGeo + Haversine + 掉队 | 3 | [x] |
| L-05 | WS Hub / Room / Fanout / Batcher / RateLimit | 3 | [x] |
| L-06 | 照片上传 + 缩略图 + 集合通知 | 3 | [x] |
| L-07 | GPSSimulator + /metrics | 3 | [x] |
| L-08 | 核心引擎单元测试 | 3 | [x] |
| U-01 | DesignSpec + 设计系统 | 2 | [x] |
| U-02 | 登录/队伍/路书编辑器 | 2 | [x] |
| U-03 | 实时大屏 + 集合 + 掉队 | 2 | [x] |
| U-04 | 照片墙 | 2 | [x] |
| Q-01 | Playwright + API smoke（Mock，¥0） | 4 | [x] |
| D-05 | Auditor + /learn | 5 | [x] |

---

## 5. 端口（Dev 随机，`/deploy` 再收敛）

| 服务 | 宿主端口 | 容器端口 |
|---|---|---|
| frontend-user | 27181 | 80 |
| backend | 27182 | 8080 |
| postgres | 27183 | 5432 |
| redis | 27184 | 6379 |

二次探测：27181–27184 于 2026-08-23 17:30 均为 FREE。
