# QA Record

## Round 1 · 2026-08-23 17:45 GMT+8

**Cost**: ¥0（Mock / 离线：GPS_PROVIDER=sim，ROUTE_PROVIDER=haversine，无计费 API）

### 执行环境
- `docker compose up --build -d` 成功，frontend 27181 / backend 27182 均 healthy
- 测试均在 compose 网络内：`docker compose run --rm qa`、`docker compose run --rm e2e`

### 结果
| 项 | 结果 |
|---|---|
| Docker Build | PASS |
| Health Check | PASS `service=gotravel tz=Asia/Shanghai` |
| Auth + 种子路书 + 里程 | PASS |
| GPSSimulator start | PASS |
| /metrics | PASS |
| Playwright e2e_flow | PASS（登录 → 营地 → 编辑路书，2.4s） |
| 浏览器手测 | PASS：登录、路书地图折线、实时大屏模拟队员、照片墙空态 |

### 日志摘要
```
SMOKE_OK
✓ e2e_flow.spec.ts:3:5 › login and open route book (1.4s)
1 passed (2.4s)
```

宿主 `go test ./...`：geo / routegraph / service / ws 通过（含 R-Tree 1000 组对照、DAG 50 节点环检测）。

无失败，进入 Phase 5。
