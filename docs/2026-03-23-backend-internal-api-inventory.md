# StrideWise 后端内部接口清单（代码现状）

更新时间：2026-03-23  
来源：`backend/internal/server/http.go`

## 1. 通用约定

- 鉴权：全部接口经过 `X-Internal-Token` 校验（统一中间件）。
- 响应：统一 Envelope。
  - `data`: 业务数据（成功时）
  - `error`: `{code, message}`（失败时）
  - `meta`: `{request_id, timestamp, fallback_reason, confidence}`
- 常见错误码：`bad_request` / `not_found` / `conflict` / `dependency_unavailable` / `internal_error`
- 时间格式：
  - 日期：`YYYY-MM-DD`
  - 日期时间：`YYYY-MM-DD HH:MM:SS`
  - `from/to/as_of` 支持 `YYYY-MM-DD` 或 `YYYY-MM-DD HH:MM:SS`

## 2. 接口清单

### 2.1 系统接口

#### `GET /internal/health`
- 入参：无
- 出参 `data`：
  - `status: string`（固定 `ok`）

#### `GET /internal/metrics`
- 入参：无
- 出参 `data`：
  - `metrics: string`

---

### 2.2 同步任务

#### `POST /internal/v1/sync/jobs`
- 入参 `body`：
  - `user_id: string`（必填）
  - `source: string`（必填，当前支持 `keep/strava/garmin/nike/gpx/tcx/fit`）
- 出参 `data`：
  - `job_id: string`
  - `status: string`（`queued`）

#### `GET /internal/v1/sync/jobs/{job_id}`
- 入参 `path`：`job_id`
- 出参 `data`：
  - `job_id: string`
  - `user_id: string`
  - `source: string`
  - `status: string`
  - `retry_count: int`
  - `error_message: string`

#### `POST /internal/v1/sync/jobs/{job_id}/retry`
- 入参 `path`：`job_id`
- 出参 `data`：
  - `job_id: string`
  - `status: string`（`queued`）
  - `retry_count: int`

---

### 2.3 用户档案

#### `POST /internal/v1/user/profile`
- 入参 `body`（必填/约束）：
  - `user_id: string`
  - `gender: string`
  - `age: int (>0)`
  - `height_cm: int (>0)`
  - `weight_kg: int (>0)`
  - `resting_hr: int`（可选；若 >0 需在 30~120）
  - `goal_type: string`
  - `goal_cycle: string`
  - `goal_frequency: int (>0)`
  - `goal_pace: string`
  - `running_years: string`（`0/<1/1-3/3+`）
  - `weekly_sessions: string`（`0-1/2-3/4+`）
  - `weekly_distance_km: string`（`0-5/5-15/15-30/30+`）
  - `longest_run_km: string`（`0/3/5/10/21`）
  - `recent_discomfort: string`（`yes/no`）
  - `location_lat: number`（必填，-90~90）
  - `location_lng: number`（必填，-180~180）
  - `country/province/city: string`（必填）
  - `location_source: string`（`geo/manual`）
- 禁止字段：`fitness_level`、`ability_level`
- 额外动作：成功后会异步触发能力层级计算任务
- 出参 `data`：当前用户档案对象（`storage.UserProfile`）

#### `GET /internal/v1/user/profile?user_id=...`
- 入参 `query`：`user_id`（必填）
- 出参 `data`：用户档案对象（`storage.UserProfile`）

---

### 2.4 天气

#### `POST /internal/v1/weather/snapshot`
- 入参 `body`：
  - `user_id: string`（必填）
  - `date: string`（可空；空时使用当天 UTC 日期）
- 逻辑：按用户位置拉取天气，计算风险并落库
- 出参 `data`：
  - `user_id: string`
  - `date: string`（`YYYY-MM-DD`）
  - `risk_level: string`（`green/yellow/red`）

#### `GET /internal/v1/weather/risk?user_id=...&date=...`
- 入参 `query`：
  - `user_id: string`（必填）
  - `date: string`（可空；空时当天 UTC 日期）
- 出参 `data`：
  - `user_id: string`
  - `date: string`
  - `risk_level: string`

---

### 2.5 训练记录

#### `POST /internal/v1/training/logs`
- 入参 `body`：
  - `user_id: string`（必填）
  - `training_type: string`（枚举：`轻松跑/有氧跑/间歇跑/长距离`；其他值会归类为 `custom`）
  - `training_type_custom: string`（可选；有值时优先）
  - `start_time: string`（`YYYY-MM-DD HH:MM:SS`）
  - `duration: string`（`HH:MM:SS`）
  - `distance_km: number (>0)`
  - `pace: string`（`mm'ss''`，例 `05'30''`）
  - `rpe: int`（1~10）
  - `discomfort: bool`
- 约束：与已有手动记录/第三方活动时间区间重叠会返回冲突
- 额外动作：成功后触发训练重算 + 能力层级异步任务
- 出参 `data`：
  - `log_id: string`
  - `job_id: string`（训练重算任务）

#### `GET /internal/v1/training/logs?user_id=...&from=...&to=...`
- 入参 `query`：
  - `user_id: string`（必填）
  - `from: string`（可选）
  - `to: string`（可选）
- 出参 `data`：训练记录数组（`storage.TrainingLog[]`）
  - 字段含：`log_id/user_id/source/training_type/training_type_custom/start_time/duration_sec/distance_km/pace_str/pace_sec_per_km/rpe/discomfort/deleted_at/created_at/updated_at`

#### `PUT /internal/v1/training/logs/{log_id}`
- 入参：`path.log_id` + 与创建同结构 `body`
- 约束：仅允许修改 `source=manual` 的记录
- 额外动作：成功后触发训练重算 + 能力层级异步任务
- 出参 `data`：
  - `log_id: string`
  - `job_id: string`

#### `DELETE /internal/v1/training/logs/{log_id}`
- 入参：`path.log_id`
- 约束：仅允许删除 `source=manual` 的记录；删除为软删除
- 额外动作：成功后触发训练重算 + 能力层级异步任务
- 出参 `data`：
  - `log_id: string`
  - `job_id: string`

---

### 2.6 每日建议

#### `POST /internal/v1/recommendations/generate`
- 入参 `body`：
  - `user_id: string`（必填）
- 约束：若用户能力层级缺失，会触发能力层级计算并返回 `ability_level_not_ready`
- 出参 `data`：
  - `rec_id: string`
  - `user_id: string`
  - `recommendation_date: string`（`YYYY-MM-DD`）
  - `created_at: datetime`
  - `input_json: object`
  - `output_json: object`
  - `override_json: object`
  - `risk_level: string`
  - `is_fallback: bool`
  - `ai_provider: string`
  - `ai_model: string`
  - `prompt_version: string`
  - `engine_version: string`
- `meta` 可能包含：
  - `fallback_reason: ai_unavailable|safety_override|...`
  - `confidence: number`

#### `GET /internal/v1/recommendations/latest?user_id=...`
- 入参 `query`：`user_id`（必填）
- 出参 `data`：同 `generate`

#### `POST /internal/v1/recommendations/{rec_id}/feedback`
- 入参：
  - `path.rec_id: string`
  - `body.user_id: string`（必填）
  - `body.useful: string`（必填）
  - `body.reason: string`（可选）
- 额外动作：可触发个性化参数重算任务
- 出参 `data`：
  - `rec_id: string`
  - `job_id: string`（可选，个性化重算任务）

---

### 2.7 基线与趋势

#### `GET /internal/v1/baseline/current?user_id=...`
- 入参 `query`：`user_id`（必填）
- 出参 `data`：`storage.BaselineCurrent`
  - `user_id/computed_at/data_sessions_7d/acute_load_srpe/chronic_load_srpe/acwr_srpe/acute_load_distance/chronic_load_distance/acwr_distance/monotony/strain/pace_avg_sec_per_km/pace_low_sec_per_km/pace_high_sec_per_km/status`

#### `GET /internal/v1/baseline/history?user_id=...&from=...&to=...`
- 入参 `query`：
  - `user_id`（必填）
  - `from/to`（可选）
- 出参 `data`：`storage.BaselineHistory[]`

#### `GET /internal/v1/training/summaries?user_id=...&from=...&to=...`
- 入参 `query`：
  - `user_id`（必填）
  - `from/to`（可选）
- 出参 `data`：训练总结数组
  - `summary_id/user_id/source_type/source_id/log_id/completion_rate/intensity_match/recovery_advice/anomaly_notes/performance_notes/next_suggestion/deleted_at/created_at/updated_at`

#### `GET /internal/v1/trends?user_id=...&window=7d|30d&as_of=...`
- 入参 `query`：
  - `user_id: string`（必填）
  - `window: string`（必填，`7d` 或 `30d`）
  - `as_of: string`（可选）
- 出参 `data`：`trend.TrendResult`
  - `window_start/window_end`
  - `summary`: `sessions/distance_km/duration_sec/avg_pace_sec_per_km/avg_rpe/summary_count/completion_rate_dist/intensity_match_dist/recovery_advice_tags/acwr_srpe/acwr_distance/monotony/strain`
  - `series[]`: `date/sessions/distance_km/duration_sec/avg_pace_sec_per_km/avg_rpe`

---

### 2.8 训练反馈

#### `POST /internal/v1/training/feedback`
- 入参 `body`：
  - `user_id: string`（必填）
  - `source_type: string`（必填，`log/activity`）
  - `source_id: string`（必填）
  - `content: string`（必填）
- 额外动作：可触发个性化参数重算任务
- 出参 `data`：
  - `feedback_id: string`
  - `job_id: string`（可选，个性化重算任务）

## 3. 目前未暴露给前端的能力（代码层存在）

- 恢复评分 `recovery_scores` 已在推荐流程中计算并落库，但当前无独立查询接口。
- 个性化参数 `user_personalization_params` 已落库并用于建议生成，但当前无独立查询接口。

