# API Contracts（Phase 3 Contract Gate）

记录时间：2026-08-23 17:45 GMT+8

本项目无第三方计费 API。Contract Gate 按提供方逐项记录：

| 提供方 | 用途 | 真实通路 | 本轮验证 | 状态 |
|---|---|---|---|---|
| Leaflet + OSM 公共瓦片 | 底图 | 浏览器直连 `tile.openstreetmap.org` | 后端不发起调用；前端内置离线网格底图降级 | **UNVERIFIED（无 Key，前端运行时探测）** |
| Haversine RouteProvider | 里程 | 本地公式 | 单元测试：北京–上海误差 < 0.5% | **verified** |
| OSRM RouteProvider | 道路里程 | `OSRM_BASE_URL` HTTP | 未配置实例，调用返回明确错误，不编造里程 | **UNVERIFIED — no key/instance** |
| Redis GEO | 空间索引 | `GEOADD/GEOPOS` | Redis 健康则可选；默认 rtree，挂了自动降级 | **verified（本地 redis 或降级）** |
| 手写 R-Tree | 空间索引 | 进程内 | 200 组随机 vs 暴力扫描一致 | **verified** |
| GPSSimulator | 队员轨迹 | 沿路书折线插值 | `GPS_PROVIDER=sim` 默认 | **verified（Mock Provider，真实路径为 live WS 上报）** |

切换开关见 `README.md` §7（`/deploy` 阶段补齐）。当前环境变量：

- `GEO_INDEX_BACKEND=rtree|redis`
- `ROUTE_PROVIDER=haversine|osrm` + `OSRM_BASE_URL`
- `GPS_PROVIDER=sim|live`
