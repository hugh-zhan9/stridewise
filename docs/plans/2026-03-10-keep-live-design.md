# StrideWise Keep 逆向登录采集设计 V1

> 目标：在保持 running_page 行为一致的前提下，支持 Keep 逆向登录拉取与标准化入库。

## 文档版本
- 当前版本：v1.0.0
- 发布日期：2026-03-10
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v1.0.0 | 2026-03-10 | 首次发布 Keep 逆向登录采集设计（完全对齐 running_page）。 |

## 1. 设计约束
1. **严格对齐** `running_page/run_page/keep_sync.py` 的登录与拉取逻辑。
2. **不落库、不缓存凭证/Token**，凭证仅来自配置文件（docker-compose 注入）。
3. 仅支持 Keep（不扩展到其他来源）。
4. 保留 `sleep(1)` 的“spider rule”节流。

## 2. 架构与组件
1. 新增 Keep 逆向登录连接器（替代 Keep 离线解析逻辑或作为默认实现）。
2. 连接器内实现：
   - 登录：`POST https://api.gotokeep.com/v1.1/users/login`
   - 列表拉取：`GET https://api.gotokeep.com/pd/v3/stats/detail?...`
   - 详情拉取：`GET https://api.gotokeep.com/pd/v3/{sport_type}log/{run_id}`
3. User-Agent、Header、请求参数与 running_page 一致。

## 3. 数据解析与标准化
1. 复刻 running_page 解码流程：
   - base64 解码 -> AES-CBC 解密（固定 key/iv）-> zlib 解压 -> JSON。
2. `geoPoints` 解码后：
   - `TRANS_GCJ02_TO_WGS84 = true`（与 running_page 一致）。
3. 轨迹 polyline：基于解码轨迹点生成 `summary_polyline`。
4. 字段映射以 running_page 为准：
   - `start_date` / `start_date_local`
   - `distance` / `moving_time`
   - `average_heartrate` / `elevation_gain`
   - `summary_polyline`

## 4. 增量与断点
1. 拉取列表按 `lastTimestamp` 递进。
2. 结合 `sync_checkpoint.last_synced_at_utc` 做增量过滤。
3. 任务成功后将 checkpoint 更新为本次最大 `start_time`。

## 5. 配置
新增配置项（明文）：
```yaml
keep:
  phone_number: "<手机号>"
  password: "<密码>"
```

## 6. 错误处理
1. 登录失败：记录 `sync_errors`，任务 `failed`。
2. 拉取失败：记录 `sync_errors`，任务 `failed`。
3. 单条记录解析失败：跳过，不阻断任务。

## 7. 验收标准
1. 与 running_page 相同账号数据拉取数量一致。
2. 增量同步不重复、不遗漏。
3. 失败任务可重试并恢复。
