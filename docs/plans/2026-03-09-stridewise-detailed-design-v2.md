# StrideWise 详细设计 V2

> 本版重点修复 V1 的可实施性缺口，并明确“AI分析建议 + 规则安全约束”的混合架构。

## 文档版本
- ~~当前版本：v2.1.0~~
- ~~当前版本：v2.2.0~~
- 当前版本：v2.3.0
- 发布日期：2026-03-09
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v2.3.0 | 2026-03-09 | 文档命名统一为 StrideWise；更新关联文档路径为 StrideWise 命名；补充“当前阶段全部接口按内部接口治理”说明。 |
| v2.2.0 | 2026-03-09 | 对齐最新技术选型：后端落地为 Kratos+Ent+PostgreSQL+Asynq+Redis；新增 Docker Compose 单节点部署约束；明确当前阶段不启用完整用户鉴权。 |
| v2.1.0 | 2026-03-09 | 新增版本管理区块；关联文档说明改为“双向引用 + 附录定位”。 |
| v2.0.0 | 2026-03-09 | 首次发布 V2，采用“AI建议 + 规则兜底”的混合决策架构。 |

## 0. 关联文档
- ~~接口与 Schema 详细规范：`docs/plans/2026-03-09-running-ai-openapi-and-json-schema.md`~~
- ~~接口与 Schema 详细规范（详细设计附录）：`docs/plans/2026-03-09-running-ai-openapi-and-json-schema.md`~~
- 接口与 Schema 详细规范（详细设计附录）：`docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`
- ~~文档关系：本文件定义系统与模块设计；关联文档定义 API 合同与 AI 输入输出契约。~~
- 文档关系：本文件定义系统、模块与数据设计；附录文档定义 OpenAPI 合同与 AI 输入输出 Schema。

## 1. 设计原则
- 安全第一：任何 AI 输出不得突破安全边界。
- 可解释：建议必须可追溯到输入与规则命中。
- 可回退：AI 不可用时系统仍可输出保守建议。
- 可运营：支持规则版本化、灰度与回滚。

## 2. 总体架构（混合决策）

### 2.1 分层
1. 客户端（App/Web）
- 建档、查看建议、记录训练、提交反馈

2. 应用层（Backend API）
- ~~Profile Service~~
- ~~Training Service~~
- ~~Weather Service~~
- ~~Recommendation Orchestrator~~
- ~~Feedback Service~~
- Kratos API Service（Profile/Training/Weather/Recommendation/Feedback）
- Asynq Worker（采集、标准化、异步建议任务）
- Scheduler（单节点轻量调度，可与 Worker 合并）

3. 决策层（Hybrid Decision Engine）
- Rule Engine（硬约束与兜底）
- AI Advisor（LLM分析与建议生成）

4. 数据层
- PostgreSQL（业务数据）
- Redis（缓存/幂等）
- ~~Queue（异步导入任务）~~
- Asynq + Redis（异步任务队列）
- Metrics/Logs/Trace（观测）

### 2.2 工程实现约束（已确认）
- 后端框架：Kratos
- ORM：Ent
- 数据库：PostgreSQL（不使用 SQLite）
- 部署方式：Docker Compose 一键部署（单节点）
- 鉴权策略：当前阶段不启用完整用户鉴权；内部接口使用 `X-Internal-Token` 保护
- API 暴露策略：当前阶段全部 API 按内部接口治理

### 2.3 决策流程
1. Orchestrator 聚合输入（天气、近7天负荷、基线、恢复状态）。
2. Rule Engine 先计算风险等级与不可违反约束（硬门禁）。
3. 调用 AI Advisor 生成候选建议（结构化 JSON）。
4. Rule Engine 对 AI 输出做二次校验与必要修正。
5. 持久化建议与解释依据，返回前端。
6. 若 AI 超时/失败：进入 Rule-only 保守建议模式。

## 3. “规则引擎”定义（非写死建议）
规则引擎不是“把所有建议写死”，而是：
- 定义不可突破的安全边界（如红色天气禁止户外高强度）。
- 定义冲突优先级（安全 > 恢复 > 天气 > 目标 > 偏好）。
- 定义兜底策略（AI失败时最小可用建议）。

AI 负责“个性化分析与建议生成”，规则负责“安全约束、合规与稳定性”。

## 4. 模块设计

### 4.1 用户与目标模块
- 字段：gender, age, height_cm, weight_kg, timezone, goal_type, goal_target。
- 新用户：问卷初始化 `baseline_current`。

### 4.2 天气模块
- 统一字段：temp, feels_like, humidity, wind_speed, precipitation_prob, aqi, uv_index。
- 增加 location（经纬度/城市）与 timezone，确保建议时间正确。
- 风险阈值由 `rule_config` 配置，不硬编码在代码。

### 4.3 训练记录模块
- 字段：train_date_local, start_at_utc, duration_min, distance_km, avg_pace, rpe, discomfort_flag, source。
- 幂等键：`user_id + source + source_record_id`。

### 4.4 基线模块
- `baseline_current`：用户当前能力快照（读性能、一致性）。
- `baseline_history`：每次更新留痕（追溯与分析）。

### 4.5 AI Advisor 模块
- 输入：结构化上下文 JSON（用户画像、负荷、天气、历史建议效果）。
- 输出：严格 JSON Schema：
  - should_run
  - workout_type
  - intensity_range
  - target_duration_or_distance
  - suggested_time_window
  - hydration_tip
  - clothing_tip
  - explanation[]
  - confidence
- 约束：禁止自由文本直接落地，必须过 schema 校验。

### 4.6 推荐编排器（Recommendation Orchestrator）
- 负责调用顺序、超时控制、重试、降级。
- 超时策略：AI 3s 超时，最多重试 1 次。
- 降级策略：走 Rule-only 模板建议并标记 `is_fallback=true`。

### 4.7 总结与反馈模块
- 训练总结落表，支持复盘和后续模型评估。
- 反馈一条建议仅允许一次有效提交（幂等约束）。

## 5. 数据库设计（V2）

### 5.1 user_profile
- id (PK)
- gender, age, height_cm, weight_kg
- timezone, default_location
- goal_type, goal_target
- ability_level
- created_at, updated_at

### 5.2 baseline_current
- user_id (PK)
- pace_zone_json, weekly_volume_range_json, recovery_level
- updated_at

### 5.3 baseline_history
- id (PK), user_id (FK)
- pace_zone_json, weekly_volume_range_json, recovery_level
- change_reason, source_event_id
- created_at

### 5.4 training_log
- id (PK), user_id (FK)
- source, source_record_id
- train_date_local, start_at_utc
- train_type, duration_min, distance_km, avg_pace
- rpe, discomfort_flag
- is_override, override_reason
- created_at, updated_at
- UNIQUE(user_id, source, source_record_id)

### 5.5 weather_snapshot
- id (PK), user_id (FK)
- snapshot_date_local, timezone, location
- temp, feels_like, humidity, wind_speed
- precipitation_prob, aqi, uv_index
- risk_level
- provider, provider_version
- created_at

### 5.6 recommendation
- id (PK), user_id (FK)
- rec_date_local, timezone
- status (draft/active/consumed/expired)
- should_run, workout_type, intensity_range, target_volume
- suggested_time_window, risk_level
- explanation_json
- is_fallback
- engine_version, prompt_version, model_name
- created_at, updated_at
- UNIQUE(user_id, rec_date_local)

### 5.7 training_summary
- id (PK), training_log_id (FK), user_id (FK)
- plan_match_score, completion_score
- recovery_tip, summary_json
- created_at

### 5.8 feedback
- id (PK), recommendation_id (FK), user_id (FK)
- usefulness, reason
- created_at
- UNIQUE(recommendation_id, user_id)

### 5.9 import_job
- id (PK), user_id (FK), provider
- status (queued/running/success/failed)
- retry_count, error_code, error_message
- started_at, finished_at, created_at

### 5.10 rule_config
- id (PK), scene, key, value_json
- version, enabled
- created_at

## 6. API 设计（V2）

### 6.1 用户与基线
- `POST /api/v1/profile/init`
- `PUT /api/v1/profile/goal`
- `GET /api/v1/profile/baseline`

### 6.2 训练与导入
- `POST /api/v1/training/logs`
- `GET /api/v1/training/logs`
- `POST /api/v1/training/import/{provider}/jobs`
- `GET /api/v1/training/import/jobs/{jobId}`

### 6.3 建议
- `GET /api/v1/recommendations/today`
- `POST /api/v1/recommendations/generate`（内部）
- `POST /api/v1/recommendations/{id}/consume`

### 6.4 总结与反馈
- `GET /api/v1/training/{id}/summary`
- `POST /api/v1/recommendations/{id}/feedback`

### 6.5 观测与运维
- `GET /internal/health`
- `GET /internal/metrics`（仅内网）

## 7. 状态机

### 7.1 recommendation.status
- draft：生成中
- active：当日可执行
- consumed：用户已执行训练
- expired：当日结束未执行

转换：
- draft -> active（生成成功）
- active -> consumed（提交 consume 或训练匹配）
- active -> expired（跨日任务）

### 7.2 import_job.status
- queued -> running -> success/failed
- failed 可按策略重试（最多 N 次）

## 8. 规则与 AI 协作细节

### 8.1 规则优先级
1. 安全规则
2. 恢复规则
3. 天气规则
4. 目标规则
5. 偏好规则

### 8.2 AI 输出校验
- JSON Schema 校验失败：直接降级 Rule-only。
- 命中安全冲突：规则覆盖 AI 输出并记录 `override_reason`。

### 8.3 Prompt 版本化
- recommendation 记录 `prompt_version`、`model_name`、`engine_version`。
- 支持灰度切换不同 prompt 版本。

## 9. 异常与降级
- AI 接口超时/失败：fallback 保守建议（低强度或休息）。
- 天气 API 失败：最近一次快照 + 风险上调一级（保守）。
- 关键字段缺失：返回低置信度建议并提示补充数据。
- 导入失败：异步重试 + 用户可手动补录。

## 10. 可观测与SLO
- SLI：建议生成成功率、P95耗时、fallback率、规则覆盖率、反馈提交成功率。
- SLO（MVP）：
  - 建议生成成功率 >= 99.5%
  - P95 建议生成耗时 <= 1500ms（不含AI超时降级路径）
  - 反馈提交成功率 >= 99%

## 11. 安全与合规
- 明示“非医疗建议”。
- 最小权限访问第三方数据，支持授权撤销与数据删除。
- 关键决策链路全审计，保留输入摘要与规则命中。
- 第三方采集凭证（token）必须加密存储，禁止明文落库。

## 12. MVP 范围确认
- AI 驱动建议：是（主路径）
- 规则引擎：是（安全门禁与兜底）
- 第三方导入：首期 1 个 provider
- 多模型/复杂个性化学习：不在 MVP
- 当前阶段用户鉴权：否（仅单用户/受控环境）
