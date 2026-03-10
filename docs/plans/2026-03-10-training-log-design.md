# StrideWise 手动训练记录模块设计文档

## 文档版本
- 当前版本：v0.1.0
- 发布日期：2026-03-10
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v0.1.0 | 2026-03-10 | 初版：定义手动训练记录模型、接口、冲突规则、软删除与异步重算。 |

## 1. 背景与目标
手动训练记录是 P0 核心链路之一，需支持新增、更新、软删除，并在变更后触发能力基线与建议刷新、总结/反馈回滚。

目标：
- 支持手动训练记录 CRUD（更新/删除仅限 manual）。
- 时间冲突严格拒绝（与手动记录及第三方活动区间重叠即冲突）。
- 变更后统一异步触发重算任务（不阻塞接口）。

## 2. 范围与边界
范围内：
- 手动训练记录存储与接口
- 冲突校验与软删除
- 异步重算任务投递（占位入口）

范围外：
- 能力基线与建议策略的实际算法（仅提供任务入口）
- 第三方记录的编辑/删除

## 3. 数据模型（TrainingLog）
表：`training_logs`（仅手动记录）

字段：
- `log_id`（UUID）
- `user_id`
- `source`（固定 `manual`）
- `training_type`（枚举或 `custom`）
- `training_type_custom`（可选）
- `start_time`（timestamp，无时区，精确到秒）
- `duration_sec`（由 `HH:MM:SS` 解析）
- `distance_km`（numeric）
- `pace_str`（原始字符串 `05'30''`）
- `pace_sec_per_km`（int）
- `rpe`（1-10）
- `discomfort`（bool）
- `deleted_at`（软删除）
- `created_at` / `updated_at`

规则：
- `training_type_custom` 不为空时，`training_type` 固定为 `custom`。
- 删除为软删除，不支持恢复。

## 4. 接口设计（内部）
- `POST /internal/v1/training/logs` 新增手动记录
- `PUT /internal/v1/training/logs/{log_id}` 更新手动记录
- `DELETE /internal/v1/training/logs/{log_id}` 软删除
- `GET /internal/v1/training/logs?user_id=...&from=...&to=...` 查询手动记录（不含 deleted）

请求字段：
- `user_id`
- `training_type` / `training_type_custom`
- `start_time`（`YYYY-MM-DD HH:MM:SS`）
- `duration`（`HH:MM:SS`）
- `distance_km`
- `pace`（`05'30''`）
- `rpe`、`discomfort`

返回字段：
- 含 `log_id`、解析后的 `duration_sec`、`pace_sec_per_km`、`created_at`/`updated_at`。

## 5. 冲突判定（拒绝手动记录）
判定区间：
- `start_time` 到 `start_time + duration_sec`

冲突规则：
- 与 **手动记录**（未删除）区间重叠 → 拒绝
- 与 **第三方活动**（`activities.start_time_local + moving_time_sec`）区间重叠 → 拒绝
- 使用 SQL `OVERLAPS` 判定
- 返回 `409 Conflict`

## 6. 校验规则
- 仅允许 `manual` 更新/删除
- `distance_km > 0`、`duration > 0`
- `pace` 必须可解析
- `rpe` 在 1-10
- `start_time` 格式合法
- `training_type` 若非枚举则要求 `training_type_custom` 非空
- 校验失败返回 `400`

## 7. 异步重算机制
新增表 `async_jobs`：
- `job_id`
- `job_type`
- `user_id`
- `payload_json`
- `status`
- `retry_count`
- `error_message`
- `created_at` / `updated_at`

触发：
- 新增/更新/删除训练记录后投递 `training_recalc` 任务

Worker 流程：
1. 标记 `running`
2. 依次调用 Baseline/Recommendation/Summary 占位入口
3. 标记 `success` 或 `failed`（失败写入 `error_message`）

## 8. 测试策略
- Storage：新增/更新/软删、冲突检测、软删不参与冲突
- Server：参数校验、冲突返回 409、只允许 manual 更新/删
- 任务：入队后 `async_jobs` 状态正确
