# StrideWise 离线导入设计 V1

> 目标：按 running_page 的“单一 activities.json + 可选文件目录”模式，完成离线导入的第一版实现，服务于阶段B的多来源接入。

## 文档版本
- 当前版本：v1.0.0
- 发布日期：2026-03-10
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v1.0.0 | 2026-03-10 | 首次发布离线导入设计，采用 running_page 模式（单一 activities.json）。 |

## 1. 设计约束与假设
1. 输入遵循 running_page 结构：`src/static/activities.json` 为主数据源。
2. `GPX_OUT/TCX_OUT/FIT_OUT` 目录仅作为上游生成 JSON 的产物来源，本阶段不解析文件内容。
3. 单次同步只绑定一个 `source`（Keep/Strava/Garmin/Nike/GPX/TCX/FIT），由调用方保证不混源。
4. 若 `activities.json` 混合多来源，系统无法区分来源，可能导致幂等冲突。

## 2. 架构与组件
1. `connector/common`：通用 JSON 解析器，读取 `activities.json`，输出 `FetchResult`。
2. `connector/keep|strava|garmin|nike|gpx|tcx|fit`：仅负责设置来源与调用通用解析器。
3. `sync/processor`：读取 checkpoint -> 调用 connector -> 标准化保存 -> 更新 checkpoint。

## 3. 数据流
1. 创建 `sync_job`（指定 user_id + source）。
2. 读取 `sync_checkpoint` 获取 `last_synced_at_utc`。
3. 解析 `activities.json`，用时间过滤增量数据。
4. 生成 `RawActivity` 与 `CanonicalActivity`，写入 `raw_activities` / `activities`。
5. 更新 `sync_checkpoint`（最大时间）。
6. 任务置为 `success` 或 `failed`。

## 4. 字段映射（运行页 JSON -> RawActivity）
| JSON 字段 | RawActivity 字段 | 说明 |
| --- | --- | --- |
| run_id | source_activity_id | 作为幂等键的一部分 |
| name | name | 活动名称 |
| distance | distance_m | 单位米 |
| moving_time | moving_time_sec | 解析为秒 |
| start_date_local | start_time | 优先使用 |
| start_date | start_time | 当 local 解析失败时回退 |
| summary_polyline | summary_polyline | 轨迹摘要 |
| 其他字段 | raw | 原样保留 |

## 5. 增量规则
1. 优先使用 `start_date_local` 解析时间。
2. 解析失败时回退 `start_date`。
3. `start_time > last_synced_at_utc` 才纳入本次结果。
4. `checkpoint.last_synced_at_utc` 取本次最大 `start_time`。

## 6. 错误处理
1. JSON 文件缺失或解析失败：记录 `sync_errors`，任务失败。
2. 单条记录解析失败：跳过记录，不阻断任务。
3. 若解析后无新数据：任务仍成功，checkpoint 不变。

## 7. 配置约定
1. 每个 source 在配置中指定 `data_file`。
2. 运行时仅执行单一来源的同步任务。

## 8. 验收与测试
1. `activities.json` 增量过滤正确。
2. 解析失败不会导致程序崩溃。
3. checkpoint 更新准确，不回退。
4. 触发一次同步后，`sync_jobs` 状态为 success。
