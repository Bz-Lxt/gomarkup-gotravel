# GoTravel Design Spec

审美方向：**夜徒制图（Night Trail Cartography）**  
给驴友看的墨绿地图书桌，而不是通用 SaaS 仪表盘。

## 调色

| Token | Hex | 用途 |
|---|---|---|
| ink | `#0E1612` | 主背景 |
| moss | `#2F6F4E` | 主行动 / 在线 |
| lantern | `#E8A04A` | 强调、集合、弱网 |
| paper | `#F3E8D4` | 正文与卡片 |
| clay | `#C46A2B` | 打卡点 / 警告 |
| dusk | `#1A2A22` | 抬升面板 |
| offline | `#6B7280` | 离线描边 |

路径点：STOP=moss，CHECKIN=lantern，LODGING=clay。

## 字体

- 标题：Fraunces（衬线，带一点旅途手账气质）
- 正文：Outfit
- 中文回退：Noto Serif SC / Noto Sans SC

禁止 Inter / Roboto / 紫渐变白底。

## 组件

- 页面容器 `w-full`，仅登录卡允许居中限宽
- 自定义 Modal / Toast（可关，5s 消失）
- 全局 `select` 自定义箭头
- 地图：深色滤镜 OSM；失败时切网格坐标底图
- 照片墙：透视层叠 + hover 抬升

## 动效

入场 420ms stagger；Marker 用 rAF 线性插值，目标 ≥30fps。
