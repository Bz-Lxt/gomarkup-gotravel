# GoTravel · Mini 旅行路书

## 1. 如何启动

```bash
docker compose up --build -d
```

浏览器打开 http://localhost:27181 。开发期端口为 27181（前端）、27182（后端）、27183（Postgres）、27184（Redis）。`/deploy` 后再收敛到 8081+。

## 2. 使用说明

使用演示账号登录后进入营地。可建队或凭邀请码加入。打开路书，在地图上点击落点，左侧列表拖拽调整顺序，里程自动重算。队长点击「开始出行」进入实时大屏，可启动模拟队员、发送紧急集合。照片墙页先点地图再上传图片。

## 3. 服务列表及API说明

| 服务 | 地址 |
|---|---|
| 用户前端 | http://localhost:27181 |
| 后端 API | http://localhost:27182/api/v1 |
| WebSocket | ws://localhost:27181/ws?token=&session_id= |
| 健康检查 | http://localhost:27182/api/v1/health |
| 扇出指标 | http://localhost:27182/api/v1/metrics |

完整协议见 `docs/API.md`。

## 4. 测试账号

| 用户名 | 密码 | 角色 |
|---|---|---|
| captain | captain123 | 队长 |
| member1 | member123 | 队员 |
| member2 | member123 | 队员 |

种子队伍「西湖夜徒小队」，邀请码 `HTK8M2`。

## 5. 题目内容

Go 全栈：多人自驾/徒步的实时足迹共享与路书编排。核心是路线依赖树、空间索引掉队检测、以及 20 人 1Hz GPS 下的 WebSocket 扇出合并。

## 6. 项目结构

```
backend/           Go 服务
frontend-user/    Vue 3 用户端
frontend-admin/   占位（需求不做后台）
frontend-mp/      占位（需求不做小程序）
tests/            API smoke + Playwright
docs/             需求与设计
```

## 7. API 模拟与切换指南

本项目所有外部能力均满足「真实通路已接线 + 本节能切换」。

| 能力 | 默认 | 真实通路 | 切换 |
|---|---|---|---|
| 队员 GPS | `GPS_PROVIDER=sim` 沿路书折线插值（含掉队者） | 浏览器/客户端经 WebSocket `type=pos` 上报真实坐标 | `GPS_PROVIDER=live` 后仅接受真实上报；模拟入口仍可用但不自动启动 |
| 里程 | `ROUTE_PROVIDER=haversine` 本地大圆距离 | `ROUTE_PROVIDER=osrm` 且设置 `OSRM_BASE_URL`，走 OSRM HTTP `/route/v1/driving` | 未配置 OSRM 时拒绝编造道路里程，回退须显式改回 haversine |
| 空间索引 | `GEO_INDEX_BACKEND=rtree` 手写 R-Tree | `GEO_INDEX_BACKEND=redis` 使用 Redis GEOADD/GEOPOS | Redis 不可用自动降级 rtree，日志会写 fallback |
| 地图底图 | 浏览器请求 OSM 公共瓦片 | 同一 Leaflet 图层 | 瓦片失败 ≥3 次自动切到离线网格坐标底图，无需 Key |

QA / CI 必须保持 `GPS_PROVIDER=sim`、`ROUTE_PROVIDER=haversine`，禁止打到计费 API。预期每轮花费 ¥0。
