# StrideWise 基线评估与训练后总结设计文档

## 文档版本
- ~~当前版本：v0.1.0~~
- 当前版本：v0.2.0
- 发布日期：2026-03-10
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v0.2.0 | 2026-03-10 | 新增训练总结 AI 生成方案、配置与失败降级策略。 |
| v0.1.0 | 2026-03-10 | 初版：基线计算、任务触发、训练总结与反馈闭环设计。 |

## 1. 背景与目标
基线评估用于衡量跑者近期负荷与能力状态，训练后总结用于形成反馈闭环。P0 要求真实计算规则、异步触发、与第三方导入兼容。

目标：
- 基线评估使用 sRPE 与距离双指标（方案C）。
- 训练记录变化与第三方同步完成后立即异步重算。
- 训练后总结覆盖更新，反馈单独存储。

## 2. 数据模型

### 2.1 BaselineCurrent
表：`baseline_current`
- `user_id`
- `computed_at`
- `data_sessions_7d`
- `acute_load_srpe`, `chronic_load_srpe`, `acwr_srpe`
- `acute_load_distance`, `chronic_load_distance`, `acwr_distance`
- `monotony`, `strain`
- `pace_avg_sec_per_km`
- `pace_low_sec_per_km`, `pace_high_sec_per_km`
- `status`（`ok/insufficient_data`）

### 2.2 BaselineHistory
表：`baseline_history`
- `baseline_id`
- `user_id`
- `computed_at`
- `trigger_type`（`training_create/update/delete/sync`）
- `trigger_ref`（log_id 或 sync_job_id）
- 其余字段与 `baseline_current` 同步存储

### 2.3 TrainingSummaries
表：`training_summaries`
- `summary_id`
- `user_id`
- `log_id`
- `completion_rate`
- `intensity_match`
- `recovery_advice`
- `anomaly_notes`
- `performance_notes`
- `next_suggestion`
- `deleted_at`
- `created_at` / `updated_at`

### 2.4 TrainingFeedbacks
表：`training_feedbacks`
- `feedback_id`
- `user_id`
- `log_id`
- `content`（自由文本）
- `created_at`

## 3. 触发机制与数据流

触发事件：
- 手动训练记录创建/更新/删除 → 立即异步触发基线重算
- 第三方同步完成 → 立即异步触发基线重算

任务类型：
- `baseline:recalc`

处理流程：
1. 汇总最近 28 天训练记录（手动 + 第三方）
2. 计算基线指标，写入 `baseline_history` 与 `baseline_current`
3. ~~生成/更新本次触发相关训练记录的 `training_summaries`~~
4. 通过 AI Provider 生成/更新本次触发相关训练记录的 `training_summaries`

## 4. 基线计算规则（真实计算）

### 4.1 训练负荷（sRPE）
- 单次负荷：`session_load = duration_min × RPE`
- 7日急性负荷：`acute_srpe = Σ last_7_days(session_load)`
- 28日慢性负荷：`chronic_srpe = Σ last_28_days(session_load) / 4`
- ACWR：`acwr_srpe = acute_srpe / chronic_srpe`（chronic>0）

### 4.2 距离负荷（补位）
- `session_distance = distance_km`
- `acute_distance / chronic_distance / acwr_distance` 同理

### 4.3 单调性与压力
- `monotony = mean(daily_load_7d) / std(daily_load_7d)`（std=0 -> 0）
- `strain = acute_srpe × monotony`（无 sRPE 时用 distance）

### 4.4 配速区间（方案A）
- `pace_avg = Σ(pace_sec_per_km × distance_km) / Σ(distance_km)`
- `pace_low = pace_avg × 0.9`
- `pace_high = pace_avg × 1.1`

### 4.5 门槛
近 7 天训练次数 < 3 → `status=insufficient_data`。

## 5. 接口设计（内部）
- `GET /internal/v1/baseline/current?user_id=...`
- `GET /internal/v1/baseline/history?user_id=...&from=...&to=...`
- `GET /internal/v1/training/summaries?user_id=...&from=...&to=...`
- `POST /internal/v1/training/feedback`（`user_id` + `log_id` + `content`）

## 6. 异常与一致性
- 数据不足：`status=insufficient_data`，仍可返回已有指标
- sRPE 缺失：ACWR_SRPE 可为空，仍返回 ACWR_DISTANCE
- 任务失败：记录 `async_jobs` 的 error_message
- ~~总结失败不影响基线入库~~
- AI 总结失败不影响基线入库，保留历史总结（若存在）并记录失败原因

## 8. 训练总结 AI 方案（新增）

### 8.1 Provider 抽象与默认实现
- 引入 AI Provider 抽象接口 `Summarizer`，支持多模型扩展
- 默认实现：OpenAI Provider（首期接入）
- 可替换实现：本地模型 Provider（后续）

### 8.2 全局模型配置
- 模型选择采用全局配置，不支持按用户覆盖
- 配置字段（示例）：
  - `ai.provider`（`openai`）
  - `ai.openai.api_key`
  - `ai.openai.base_url`
  - `ai.openai.model`
  - `ai.openai.timeout_ms`
  - `ai.openai.max_tokens`
  - `ai.openai.temperature`

### 8.3 输入输出约束
- 输入最小化：仅发送训练记录摘要、基线关键指标、最近训练频率
- 禁止传输原始轨迹、设备标识等敏感信息
- 输出必须映射到 `training_summaries` 字段

### 8.4 失败与降级
- AI 超时/失败：记录 `async_jobs.error_message`
- 若已有总结：保留历史总结不覆盖
- 若无总结：写入规则占位文本（可追溯）

### 8.5 与“无外部 API”原则的兼容
- 对外不提供公开 API
- 允许向外部 AI 供应商发起调用（OpenAI）作为内部依赖

## 7. 测试策略
- Storage：基线计算/历史保存/总结覆盖/反馈写入
- Server：接口参数校验、数据不足状态
- Worker：任务处理状态流转
