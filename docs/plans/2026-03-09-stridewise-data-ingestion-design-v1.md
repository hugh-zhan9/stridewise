# StrideWise 数据采集与标准化系统详细设计 V1

> 目标：仅复用 running_page 的“采集 + 标准化”思路，构建可支撑多用户和 AI 训练建议的生产级数据底座。

## 文档版本
- ~~当前版本：v1.0.0~~
- ~~当前版本：v1.1.0~~
- 当前版本：v1.2.0
- 发布日期：2026-03-09
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v1.2.0 | 2026-03-09 | 数据源范围新增 Keep，并确认首发 Connector 为 Keep。 |
| v1.1.0 | 2026-03-09 | 对齐最新技术选型与任务约束：补充 Kratos+Ent+PostgreSQL+Asynq+Redis+Docker Compose 单节点约束；修正 `sync_jobs` 必填字段为 `job_id + status + retry_count + error_message`。 |
| v1.0.0 | 2026-03-09 | 首次发布数据采集与标准化系统设计，覆盖架构、模型、状态机、API 与实施计划。 |

## 1. 设计目标
1. 统一接入多源运动数据（Keep/Strava/Garmin/Nike/GPX/TCX/FIT）。
2. 将异构数据标准化为统一活动模型，供后续 AI 建议服务消费。
3. 支持增量同步、重试、幂等、审计、回溯，不依赖静态 JSON 文件。

## 2. 设计范围
### 2.1 包含
- Connector 采集框架
- 标准化规则与字段映射
- 同步任务状态管理
- 数据库存储模型
- 内部查询与触发同步 API

### 2.2 不包含
- AI 建议生成
- 前端页面渲染
- 训练计划策略逻辑

## 3. 架构设计

### 3.0 实现选型约束（已确认）
- 后端框架：Kratos
- ORM：Ent
- 数据库：PostgreSQL
- 异步队列：Asynq + Redis
- 部署方式：Docker Compose 一键部署（单节点）

### 3.1 模块划分
1. `connector`：各平台采集适配器
2. `normalizer`：原始数据 -> 标准活动模型
3. `pipeline`：同步任务编排（增量、重试、断点）
4. `storage`：原始表/标准表/状态表
5. `internal-api`：供上层服务拉取活动与任务状态

### 3.2 运行流程
1. 创建 `sync_job`（指定用户、数据源、同步模式）
2. 读取 `sync_checkpoint` 获取上次断点
3. Connector 拉取分页数据（cursor/since）
4. 原始 payload 入 `raw_activities`
5. Normalizer 标准化后 UPSERT 到 `activities`
6. 更新 `sync_checkpoint` 与 `sync_job` 状态
7. 产出事件（可选）：`activity.upserted`

## 4. 标准活动模型（Canonical Activity）

```text
activity_id: string                # 内部ID（UUID）
user_id: string
source: enum[keep, strava, garmin, nike, gpx, tcx, fit]
source_activity_id: string         # 外部平台活动ID
sport_type: enum[running, cycling, hiking, walking, swimming, skiing, other]
sub_type: string|null
name: string|null
start_time_utc: datetime
start_time_local: datetime
timezone: string
moving_time_sec: int
elapsed_time_sec: int|null
distance_m: float
avg_speed_mps: float|null
avg_heartrate: float|null
elevation_gain_m: float|null
summary_polyline: string|null
location_country: string|null
location_province: string|null
location_city: string|null
data_quality_score: int            # 0-100
is_deleted: bool
created_at: datetime
updated_at: datetime
```

## 5. 标准化规则

### 5.1 时间与时区
1. 所有源时间统一存 `start_time_utc`。
2. 同时存 `start_time_local + timezone` 便于本地日历统计。
3. 无时区源按用户默认时区推断并标注 `data_quality_score` 降级。

### 5.2 单位统一
1. 距离统一米（`distance_m`）。
2. 速度统一米/秒（`avg_speed_mps`）。
3. 海拔统一米（`elevation_gain_m`）。

### 5.3 类型映射
- `Run/running` -> `running`
- `Ride/cycling` -> `cycling`
- `Hike/hiking` -> `hiking`
- `Walk/walking` -> `walking`
- 其他映射到 `other`

### 5.4 幂等键
唯一键：`(user_id, source, source_activity_id)`。

### 5.5 数据质量分级（建议）
- 100：核心字段齐全（时间、距离、时长、类型、轨迹）
- 80：缺轨迹
- 60：缺心率/海拔
- 40：时间或时区存在推断
- <40：关键字段缺失（需告警）

## 6. 数据库设计

### 6.1 raw_activities（原始落地）
- id (PK)
- user_id
- source
- source_activity_id
- payload_json
- fetched_at
- job_id
- UNIQUE(user_id, source, source_activity_id)

### 6.2 activities（标准活动）
- id (PK)
- user_id
- source
- source_activity_id
- sport_type, sub_type, name
- start_time_utc, start_time_local, timezone
- moving_time_sec, elapsed_time_sec
- distance_m, avg_speed_mps, avg_heartrate, elevation_gain_m
- summary_polyline
- location_country, location_province, location_city
- data_quality_score
- is_deleted
- created_at, updated_at
- UNIQUE(user_id, source, source_activity_id)
- INDEX(user_id, start_time_local DESC)
- INDEX(user_id, sport_type, start_time_local DESC)

### 6.3 sync_jobs（同步任务）
- id (PK)
- job_id (UNIQUE)
- user_id
- source
- mode (full/incremental)
- status (queued/running/success/failed/partial)
- retry_count
- started_at, finished_at
- fetched_count, upserted_count, failed_count
- error_code, error_message
- created_at

### 6.4 sync_checkpoints（断点）
- id (PK)
- user_id
- source
- cursor
- last_synced_at_utc
- updated_at
- UNIQUE(user_id, source)

### 6.5 sync_errors（错误审计）
- id (PK)
- job_id
- source
- source_activity_id
- error_code
- error_message
- retryable
- created_at

## 7. 同步状态机

### 7.1 Job 状态
- `queued` -> `running` -> `success`
- `running` -> `partial`（部分失败）
- `running` -> `failed`

### 7.2 重试策略
1. 可重试错误（网络超时/429/5xx）：指数退避，最多 5 次。
2. 不可重试错误（鉴权失败/参数错误）：立即失败。
3. 单条数据标准化失败不阻断全任务，记录 `sync_errors`。

## 8. Connector 规范（接口契约）

```python
class Connector:
    def fetch_activities(self, user_credential, since=None, cursor=None, limit=100):
        """返回 (items, next_cursor, has_more)"""

    def validate_credential(self, user_credential):
        """检查授权有效性"""
```

每个 provider 只负责“拉取与最小格式化”，不做业务规则处理。

## 9. 内部 API 设计

### 9.1 触发同步
- `POST /internal/v1/sync/jobs`
- 入参：`user_id, source, mode`
- 出参：`job_id, status`

### 9.2 查询任务
- `GET /internal/v1/sync/jobs/{job_id}`
- 出参：状态、计数、错误摘要

### 9.3 查询活动列表
- `GET /internal/v1/activities?user_id=...&start=...&end=...&sport_type=...&cursor=...`
- 游标分页返回标准活动数据

### 9.4 获取最后断点
- `GET /internal/v1/sync/checkpoints?user_id=...&source=...`

## 10. 错误码规范
- `SYNC_AUTH_FAILED`：授权失效
- `SYNC_RATE_LIMITED`：频控限制
- `SYNC_PROVIDER_UNAVAILABLE`：上游不可用
- `SYNC_TIMEOUT`：超时
- `NORMALIZE_INVALID_FIELD`：标准化字段异常
- `DB_UPSERT_FAILED`：写库失败

## 11. 安全与合规
1. 凭证仅存加密密文（KMS/密钥托管）。
2. 日志脱敏：不记录 access_token、cookie、手机号等。
3. 支持用户授权撤销后停止同步，并可删除关联数据。
4. 仅内部网络可访问 `/internal/*`。

## 12. 观测指标（SLI）
1. 同步成功率（job success / total）
2. 同步端到端耗时 P95
3. 每源错误率（按 source 维度）
4. 数据新鲜度（`now - last_synced_at_utc`）
5. 单条标准化失败率

## 13. 实施计划（建议 3 周）

### 第1周
- 建库建表：`raw_activities/activities/sync_jobs/sync_checkpoints/sync_errors`
- 搭建 pipeline 与状态机
- ~~接入第一个 Connector（建议 Strava）~~
- 接入第一个 Connector（Keep）

### 第2周
- 完成标准化规则与幂等 UPSERT
- 打通内部 API（触发任务/查任务/查活动）
- ~~接入第二个来源（建议 Garmin）~~
- 接入第二个来源（建议 Strava）

### 第3周
- 增加重试与告警
- 加入数据质量评分
- 压测与回归（增量、全量、异常场景）

## 14. 验收标准
1. 同一活动重复同步不产生重复记录。
2. 增量同步可基于 checkpoint 连续执行。
3. provider 临时故障可自动重试并最终可恢复。
4. 关键字段统一成功率 >= 99%。
5. 上层服务可通过内部 API 稳定读取标准化活动数据。

## 15. 与 running_page 的复用建议
1. 复用：各平台同步脚本思路、活动通用字段、polyline 处理思路。
2. 不复用：静态 `activities.json` 作为主数据源、前端耦合式流水线。
3. 迁移策略：先“脚本可用”，再逐步“服务化”。
