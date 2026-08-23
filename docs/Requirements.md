# GoTravel · Mini 旅行路书 — 需求规格说明书

| 项目 | 内容 |
|---|---|
| 项目代号 | GoTravel |
| 文档版本 | v1.0（PM Agent 冻结） |
| 冻结时间 | 2026-08-23 17:19 (GMT+8) |
| SOP 版本 | Alkaid-SOP v13.0 |
| 需求权威 | 本文件定义 **WHAT**；`docs/Roadmap.md` 定义 **WHEN** |
| 原始需求 | `docs/.meta/original_prompt.md`（SSOT，不可修改） |

---

## 0. PM 判废评估结论：**ACCEPT（受控接受）**

按 Alkaid-SOP v13.0 §4 判废标准逐条评估：

| # | 判废标准 | 评估结果 | 结论 |
|---|---|---|---|
| 1 | 需求不完整/模糊 | 主题明确（多人出行实时足迹 + 路书编排），三大前端模块与两大后端技术挑战均有清晰描述，无缺失附件依赖 | ✅ PASS |
| 2 | Windows 独占 | Go + Vue 3 + Docker，全链路跨平台；本机已验证 Go 1.25.12 darwin/arm64 | ✅ PASS |
| 3 | 规模评估（分级） | 估算 **11,000 – 14,000 LoC**（详见 §7），落入 10k–40k 区间 | ⚠️ **ACCEPT，但分期 Roadmap 为强制前置条件** |
| 4 | 外部依赖（智能判定） | 全部落入 Scenario A（可模拟），无 Scenario B 项。详见 §1 | ✅ PASS |
| 5 | 特殊/冷门（需付费商业软件） | Go / Vue / Leaflet / OSM / Redis / PostgreSQL 全开源免费 | ✅ PASS |

### 规模分级的强制约束（触发 v13 三级规则第二档）

本项目 **不得**在 `docs/Roadmap.md` 完成 MVP / V1 / V2 阶段边界划分前写入任何业务代码。
Phase 1 Chief Architect 必须显式产出阶段边界，否则 Phase 2 不得启动。

---

## 1. 外部依赖智能判定（v13 §4 Step 1.4）

| 依赖项 | 场景判定 | 处置策略 |
|---|---|---|
| **地图底图瓦片** | A（可模拟） | 采用 **Leaflet + OpenStreetMap 公共瓦片**，零 API Key。Docker 内置离线降级底图（纯色网格 + 坐标标尺），断网时前端自动切换，保证 `docker compose up` 后地图必可渲染 |
| **队员实时 GPS 上报** | A（可模拟） | GPS 是**队员自身上报的数据**，非第三方事实数据。提供 `GPSSimulator` 沿路书折线插值生成 N 个虚拟队员轨迹（含掉队者、含弱网抖动），用于演示与压测 |
| **道路级路径规划 / 真实驾车里程** | A（可模拟） | 默认使用 **Haversine 大圆距离 + 折线累加**（纯本地计算，零依赖）。预留 `RouteProvider` 接口，可切换至自建 OSRM 实例获取道路真实里程 |
| **照片存储** | 无外部依赖 | 本地 Docker Volume + Go 静态文件服务 + 服务端缩略图生成 |
| **紧急集合通知** | 无外部依赖 | 站内 WebSocket 强制推送 + 浏览器 Notification API，**不接短信/电话通道** |

**判定结论**：无任何依赖属于 Scenario B（实时股价 / 实时赛事 / 实时路况等不可模拟的事实数据）。
本项目虽是"出行"场景，但**不需要实时路况**——只需要队员彼此的位置，而位置由队员自己上报。

> **Mock 合法性承诺（Redline 4 · v13 Mock Legitimacy Standard）**
> 上表所有 Mock 项均满足两个条件：(a) 真实实现路径存在且已接线（接口抽象 + 环境变量开关）；(b) Mock/Real 切换方式必须在 `README.md` §7 「API 模拟与切换指南」中逐项文档化。
> 任何"静默用 Mock 顶替真实逻辑且无文档开关"的实现，Phase 5 审核直接判 FAIL。

---

## 2. 矛盾检测与技术决策裁定（v13 §4 Step 2）

原始 Prompt 中存在 3 处二选一表述与 1 处工程张力，PM 在此一次性裁定，**Phase 1–5 不得反复推翻**：

| # | 原文表述 | 冲突性质 | **PM 裁定** | 理由 |
|---|---|---|---|---|
| C-1 | "Mapbox GL / Leaflet" | 二选一，二者不可共存于同一渲染层 | **Leaflet 1.9.x + OSM** | Mapbox GL 免费额度需注册 Access Token，硬依赖外部密钥会破坏 `docker compose up` 开箱即用（Redline 1）。Leaflet 零 Key、体积小、拖拽/Marker 生态成熟 |
| C-2 | "基于 Redis Geo 或本地 R-Tree" | 二选一 | **两者都实现，接口切换** | 抽象 `geo.Index` 接口，提供 `RedisGeoIndex`（GEOADD/GEOSEARCH）与 `RTreeIndex`（手写 R-Tree）两套实现，环境变量 `GEO_INDEX_BACKEND` 切换。Redis 不可用时自动降级到内存 R-Tree，兼顾"演示技术深度"与"单容器可跑" |
| C-3 | "空间空间索引" | 明显笔误（叠词） | 按 **"空间索引"（Spatial Index）** 理解 | 无歧义 |
| C-4 | "20 人每人每秒上报 GPS" | 与户外场景真实功耗/流量存在张力（1Hz 上报对手机电量不友好） | **保留 1Hz 作为压测基线，同时实现自适应上报频率** | 该数字是需求指定的**压力挑战靶子**，必须扛住；但客户端实现"静止 0.2Hz / 移动 1Hz"的自适应策略作为工程加分项，不降低服务端基线要求 |

---

## 3. Docker 交付标准合规性检查（Redline 1）

| 检查项 | 结论 |
|---|---|
| 微信小程序豁免 | ❌ 不适用（本项目为 Web 端） |
| `docker compose up --build -d` 一键启动 | ✅ 技术栈完全支持，无手工步骤 |
| 暴露 localhost 可访问服务 | ✅ 前端 SPA（Nginx）+ 后端 REST/WebSocket，浏览器直达 |
| ARM64 / AMD64 双架构 | ✅ 基础镜像均为官方多架构：`golang:1.25-alpine`、`node:21-alpine`、`nginx:1.27-alpine`、`redis:7-alpine`、`postgres:16-alpine` |
| 时区统一 | ✅ 所有容器强制 `TZ=Asia/Shanghai`；Go 侧统一使用 GMT+8 时间工具函数，禁止裸用 `time.Now().UTC()` |

**服务清单（Dev 阶段随机端口，`/deploy` 阶段收敛至 8081+）**

| 服务 | 容器 | 职责 |
|---|---|---|
| `frontend` | nginx | Vue 3 SPA 静态托管 + `/api` `/ws` 反向代理 |
| `backend` | golang | REST API + WebSocket Hub + 扇出引擎 + 空间索引 |
| `postgres` | postgres:16 | 用户/队伍/路书/依赖树/照片元数据持久化 |
| `redis` | redis:7 | GEO 空间索引 + 在线状态 + 广播限流令牌 |

---

## 4. 功能需求（分级：P0 = MVP 必须，P1 = V1，P2 = V2）

### 4.1 账号与队伍（Team）

| ID | 优先级 | 需求 |
|---|---|---|
| F-A01 | P0 | 用户注册/登录，JWT 鉴权，头像（预置 Avatar 色块 + 首字母，无需上传） |
| F-A02 | P0 | 创建队伍，生成 6 位邀请码；队员凭码加入 |
| F-A03 | P0 | 角色区分：**队长（Leader）** 与 **队员（Member）**。队长独占：编辑路书、发紧急集合、解散队伍 |
| F-A04 | P1 | 队长转让、移除队员 |

### 4.2 路书编辑器（Route Book Editor）— 前端核心模块 1

| ID | 优先级 | 需求 |
|---|---|---|
| F-B01 | P0 | 地图上点击生成经纬度路径点（Waypoint），支持逆向坐标标注 |
| F-B02 | P0 | 路径点分类：**经停点(STOP) / 小众打卡点(CHECKIN) / 住宿点(LODGING)**，不同图标与配色 |
| F-B03 | P0 | 列表拖拽（drag-and-drop）调整顺序，地图折线实时重绘 |
| F-B04 | P0 | **自动计算总里程**：Haversine 逐段累加，实时显示总距离 / 分段距离 / 预估耗时 |
| F-B05 | P0 | 路径点增删改、批量导入 GeoJSON |
| F-B06 | P1 | **路线依赖树**：为路径点配置前置依赖（DAG），支持"必须先到 A 才能到 B"的非线性约束；服务端拓扑排序校验 + 环检测 |
| F-B07 | P1 | 路书导出为 GeoJSON / GPX |
| F-B08 | P2 | 路书模板库与一键复制 |

### 4.3 多人实时位置大屏（Live Map）— 前端核心模块 2

| ID | 优先级 | 需求 |
|---|---|---|
| F-C01 | P0 | 开启"出行实例（Trip Session）"，队员 WebSocket 接入，上报 GPS |
| F-C02 | P0 | 地图显示全体队员 Marker（头像 + 昵称），**帧间线性插值实现平滑移动**（非跳跃刷新） |
| F-C03 | P0 | **网络状态指示**：在线 / 弱网（>3s 无上报）/ 离线（>15s），Marker 边框颜色区分 |
| F-C04 | P0 | 队长**一键紧急集合通知**：全员强制弹窗 + 声音提示 + 集合坐标下发，含已读回执统计 |
| F-C05 | P0 | **掉队预警**：服务端周期性用空间索引计算离队伍质心/队首最远的队员，超阈值（默认 2km）自动推送预警给队长与本人 |
| F-C06 | P1 | 队员历史轨迹回放（时间轴拖动） |
| F-C07 | P1 | 队伍整体进度条：已完成路径点 / 总路径点，剩余里程 |
| F-C08 | P2 | 队员间一对一定向语音/文字消息 |

### 4.4 旅行手账 / 照片墙（Travel Journal）— 前端核心模块 3

| ID | 优先级 | 需求 |
|---|---|---|
| F-D01 | P0 | 在地图指定坐标点上传照片（拖拽/点选），服务端存储 + 生成缩略图 |
| F-D02 | P0 | **立体感足迹照片墙**：按拍摄坐标聚类展示，CSS 3D transform + 层叠错位 + hover 视差，点击定位到地图坐标 |
| F-D03 | P0 | 照片附带文字手账、时间戳（GMT+8）、上传者 |
| F-D04 | P1 | 地图上照片点聚合（Cluster），缩放自动展开 |
| F-D05 | P2 | 按天生成旅行日报（自动串联轨迹 + 照片） |

### 4.5 Go 后端技术挑战（**本项目的核心考核项，全部 P0**）

#### 挑战一：路线依赖树与地理信息存储

| ID | 需求 |
|---|---|
| F-E01 | 支持**几十个（≥50）**景点节点的先后依赖顺序管理，以 **DAG** 建模 |
| F-E02 | 手写**拓扑排序**（Kahn 算法）生成可行访问序列；**环检测**并精确报出环路节点链 |
| F-E03 | 依赖冲突检测：新增边若成环则拒绝，返回冲突路径 |
| F-E04 | 地理信息持久化：路径点坐标、路书折线、照片坐标统一存储；坐标合法性校验（经度 ±180 / 纬度 ±90 / 精度截断） |
| F-E05 | **空间索引双实现**：`RedisGeoIndex`（GEOADD / GEOSEARCH / GEODIST）+ **手写 R-Tree**（节点分裂采用 Guttman 二次分裂，支持 Insert / Delete / RangeSearch / kNN） |
| F-E06 | **掉队检测引擎**：基于空间索引计算队伍质心与最远队员，输出 `LaggardAlert`；支持"距队首""距质心""距下一个路径点"三种口径 |

#### 挑战二：高频 WebSocket 消息扇出优化

| ID | 需求 |
|---|---|
| F-F01 | **手写高弹性 Channel 缓冲区**：每连接独立有界 `send chan`，满时执行**丢弃最旧位置帧（Drop-Oldest）**而非阻塞 Hub 主循环 |
| F-F02 | **合并转发（Coalescing）**：以 200ms tick 为窗口，将窗口内同一队员的多帧位置**只保留最新一帧**，多队员合并为**单个批量帧**下发 |
| F-F03 | **流量限制**：单连接上行令牌桶（默认 5 msg/s，突发 10），超限丢弃并回 `RATE_LIMITED`；下行按连接背压自适应降频 |
| F-F04 | **慢消费者隔离**：连续 N 次写超时的连接强制踢下线，绝不允许单个慢客户端拖垮整个 Hub |
| F-F05 | 消息分级：位置帧（可丢弃、可合并）/ 紧急集合与掉队预警（**不可丢弃**，独立优先通道，必达） |
| F-F06 | Hub 分房间（Room = TripSession）隔离，房间内广播不影响其他房间；单进程支持 ≥100 房间 |
| F-F07 | 心跳 Ping/Pong + 断线重连 + 断连期间位置补发（客户端本地队列） |
| F-F08 | **可观测性**：暴露扇出指标（入站 QPS、出站 QPS、合并压缩率、丢帧数、踢人数、房间数、P99 广播延迟） |

### 4.6 工程规范（继承全局记忆强制项）

| ID | 需求 | 来源 |
|---|---|---|
| F-G01 | 外部数据导入/反序列化必须校验结构完整性（字段存在性、类型、边界值），GeoJSON 导入与 WS 消息解析尤其严格 | 全局记忆 [Robustness] |
| F-G02 | 统一 Logger（分级 + 结构化），禁止散落 `fmt.Println` / `console.log`；生产环境自动屏蔽 debug | 全局记忆 [Logging] |
| F-G03 | 必须提供独立 `docs/API.md`：每个端点的请求/响应示例、参数类型说明、完整错误码表（含 WebSocket 消息协议表） | 全局记忆 [Documentation] |
| F-G04 | 必须包含测试代码：后端覆盖 CRUD + **核心引擎（DAG / R-Tree / 扇出）** 单元测试；E2E 用 Playwright | 全局记忆 [Testing] |
| F-G05 | 全链路 GMT+8：容器 `TZ=Asia/Shanghai`，Go 侧统一时间工具，禁止裸 UTC | 时区规范 |

---

## 5. 可量化验收基线（v13 Acceptance Baselines · Phase 4/5 据此判定）

> 以下为**可测量**指标，非描述性文字。Phase 4 QA 必须在 Mock 模式下实测并记录到 `docs/QA_Record.md`。

### 5.1 扇出性能基线（对应需求原文"20 人每人每秒上报"）

| 指标 | 基线 | 测量方式 |
|---|---|---|
| 承载规模 | 单房间 20 连接 × 1 Hz 上报，持续 60s 无错误 | GPS Simulator 压测 |
| 端到端广播延迟 | **P99 < 500ms**，P50 < 200ms | 帧内嵌服务端时间戳比对 |
| 合并压缩率 | 相比朴素 N×N 广播，**下行消息条数降低 ≥ 80%** | 朴素基线 400 msg/s → 合并后 ≤ 80 msg/s |
| 关键消息投递 | 紧急集合 / 掉队预警 **丢失率 = 0** | 100 次发送全量核对回执 |
| 位置帧丢弃 | 允许丢弃，但**每连接连续丢弃不得 > 5 帧**（否则视为背压失控） | 指标接口读取 |
| 资源占用 | 20 连接稳态下 backend 容器 **CPU < 50%（单核）**，**内存 < 256MB** | `docker stats` |
| Goroutine 泄漏 | 压测结束 30s 后 goroutine 数回落至基线 **±10** 以内 | `/metrics` 或 pprof |
| 房间隔离 | 100 房间并发时，A 房间广播不产生 B 房间的出站消息 | 计数器断言 |

### 5.2 空间索引与掉队检测基线

| 指标 | 基线 |
|---|---|
| "最远队员"查询延迟（20 人） | **P99 < 10ms**（Redis Geo 与 R-Tree 两种后端均需达标） |
| R-Tree 正确性 | 与暴力线性扫描结果**逐条一致**（≥1000 组随机数据交叉验证） |
| Haversine 里程精度 | 与参考实现误差 **< 0.5%**（用北京–上海等已知距离校验） |
| 掉队预警触发延迟 | 队员越过阈值后 **≤ 3s** 内产生告警 |
| 掉队预警去抖 | 同一队员同一次掉队事件**不得重复告警**（冷却窗口 60s） |

### 5.3 依赖树基线

| 指标 | 基线 |
|---|---|
| 节点规模 | 单路书支持 **≥ 50 个**路径点、**≥ 100 条**依赖边 |
| 拓扑排序 | 50 节点排序耗时 **< 5ms**，结果满足全部偏序约束 |
| 环检测 | 100% 检出，且返回**具体环路节点链**（非仅布尔值） |

### 5.4 前端体验基线

| 指标 | 基线 |
|---|---|
| Marker 平滑移动 | 插值渲染 **≥ 30 FPS**（20 个 Marker 同屏） |
| 首屏加载 | 本地 Docker 环境 **< 2s** |
| 响应式 | 1920 / 1366 / 768 / 375 四档宽度无布局破损 |
| 照片上传 | 单张 ≤ 10MB，缩略图生成 **< 1s** |
| 拖拽排序 | 松手后地图折线与总里程 **< 100ms** 内更新 |

### 5.5 成本基线（v13 Cost-Safe Testing）

| 指标 | 基线 |
|---|---|
| 本项目外部计费 API | **无**（Leaflet/OSM 免费、无 LLM、无短信） |
| QA 每轮预期支出 | **¥0**（强制 Mock/离线模式） |

---

## 6. 数据模型草案（Phase 1 可细化，不可推翻主干）

```
users          (id, username, password_hash, nickname, avatar_color, created_at)
teams          (id, name, leader_id, invite_code, status, created_at)
team_members   (team_id, user_id, role, joined_at)
trips          (id, team_id, title, description, cover, total_distance_m, status, created_at)   -- 路书
waypoints      (id, trip_id, seq, name, kind[STOP|CHECKIN|LODGING], lat, lng, altitude,
                planned_stay_min, note, created_at)
waypoint_deps  (trip_id, from_waypoint_id, to_waypoint_id)                                     -- DAG 边
trip_sessions  (id, trip_id, started_at, ended_at, status)                                      -- 出行实例
position_reports (id, session_id, user_id, lat, lng, speed, accuracy, heading, reported_at)      -- 时序，分区/滚动清理
photos         (id, trip_id, session_id, user_id, lat, lng, file_path, thumb_path, caption,
                taken_at, created_at)
alerts         (id, session_id, type[RALLY|LAGGARD], payload, created_by, created_at)
alert_acks     (alert_id, user_id, acked_at)                                                    -- 已读回执
```

**关键约束**
- `waypoint_deps` 必须保证无环（应用层 DAG 校验 + 唯一索引防重边）
- `position_reports` 高频写入，需索引 `(session_id, user_id, reported_at DESC)`，并提供滚动清理策略避免无限膨胀
- 所有 `*_at` 字段以 GMT+8 语义写入

---

## 7. 规模估算（支撑 §0 第 3 条判定）

| 模块 | 文件数 | 预估 LoC |
|---|---|---|
| Go — 入口/配置/日志/DB/Redis | 7 | 700 |
| Go — model | 8 | 700 |
| Go — repository | 7 | 1,000 |
| Go — service（auth/team/trip/route/geo/photo/alert） | 7 | 1,400 |
| Go — handler + middleware | 11 | 1,300 |
| Go — **ws（hub/client/room/fanout/batcher/ratelimit）** | 6 | 1,300 |
| Go — **geo（index 接口/redisgeo/rtree/haversine/laggard）** | 5 | 1,100 |
| Go — **routegraph（dag/toposort/distance）** | 3 | 600 |
| Go — simulator（GPS Mock Provider） | 2 | 400 |
| Go — pkg（response/errors/timeutil） | 3 | 300 |
| Go — 单元测试 `*_test.go` | 10 | 1,200 |
| **Go 小计** | **69** | **≈ 10,000** |
| 前端 Vue 3 + TS（页面/组件/store/ws-client/地图封装） | ~45 | ≈ 4,000 |
| Playwright E2E | 3 | 400 |
| Docker / Nginx / SQL 迁移 / 配置 | ~10 | 400 |
| **总计** | **≈ 127** | **≈ 14,800** |

Go 文件数 **69 ≫ 34** 的下限要求；Go 代码量 **≈10,000 行**略高于用户给出的 6,500–9,000 区间，主要来自"双空间索引实现 + 完整测试覆盖"。
**处置**：R-Tree 与 Redis Geo 双实现、以及 P1/P2 功能可按 Roadmap 分期落地，MVP 阶段先交付 Redis Geo + 完整扇出引擎，使 MVP 的 Go 代码量落在 **6,500–7,500 行**的目标区间内。

总规模 ≈ 14,800 LoC → 落入 v13 第二档（10k–40k），**接受，分期 Roadmap 强制**。

---

## 8. 明确不做（防范 Redline 4 范围漂移）

以下内容**不在**本项目范围内，Phase 1–5 不得自行加入：

- ❌ 真实短信 / 电话 / 微信推送通道
- ❌ 支付、订票、酒店预订、行程消费结算
- ❌ 原生 iOS / Android App（仅交付响应式 Web；移动端浏览器可用即可）
- ❌ 实时路况、天气预报接入（Scenario B 不可模拟数据）
- ❌ 社交内容审核、AI 行程推荐、AI 图文生成
- ❌ 多租户 SaaS 计费与运营后台
- ❌ 道路级导航转向指引（TBT Navigation）

---

## 9. 交付物清单

| 交付物 | 责任阶段 |
|---|---|
| `docs/Requirements.md` | ✅ PM（本文件） |
| `docs/.meta/original_prompt.md` | ✅ PM |
| `docs/Roadmap.md`（**含 MVP/V1/V2 边界 + Phase Order 决策**） | Phase 1 Chief Architect |
| `docs/DesignSpec.md` | Phase 2 UI Agent |
| `docs/API.md`（REST + WebSocket 协议 + 错误码表） | Phase 3 Logic Agent |
| `docs/.meta/api_contracts.md` | Phase 3 Contract Gate |
| `docs/QA_Record.md`（每轮含 Cost 字段 = ¥0） | Phase 4 QA |
| `docs/AuditReport.md` | Phase 5 Auditor |
| `docs/SelfTestReport.md` + `README.md`（7 个强制章节） | `/deploy` |

---

## 10. Phase Order 建议（供 Phase 1 决策参考，非最终决定）

Alkaid-SOP v13 §4 Phase 1.5 要求显式记录构建顺序。PM 观察：

本项目前端主体是 **地图编辑器 + 实时轨迹画布**，其组件结构**强依赖**于 Waypoint / DAG / 位置帧的数据模型形状，符合 v13 "editor / timeline / canvas / visualization" 的 **Logic-First** 触发条件。

**PM 建议**：采用 **Logic-First**，即 **交换 Phase 2 与 Phase 3** —— 先落地数据模型、DAG 引擎、空间索引与 WebSocket 扇出协议，再据此构建前端。
最终决定权归 Phase 1 Chief Architect，须在 `docs/Roadmap.md` 中以一句话记录理由。

---

**需求已冻结。** 输入 `/auto` 启动 Auto-Swarm（Phase 1 → 5）。
