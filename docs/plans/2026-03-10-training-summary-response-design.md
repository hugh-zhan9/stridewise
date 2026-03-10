# StrideWise 训练总结接口响应字段设计

## 文档版本
- 当前版本：v0.1.0
- 发布日期：2026-03-10
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v0.1.0 | 2026-03-10 | 训练总结列表接口响应统一为 snake_case。 |

## 1. 背景与目标
当前 `GET /internal/v1/training/summaries` 直接返回 `storage.TrainingSummary`，JSON 字段为结构体名（驼峰），与 OpenAPI 约定的 `snake_case` 不一致，影响前后端对齐与长期演进。

目标：
- 接口响应字段严格使用 `snake_case`，与 OpenAPI 文档一致。
- 不改变存储与业务逻辑，仅调整序列化输出。

## 2. 范围与非范围
范围：
- `GET /internal/v1/training/summaries` 响应 DTO 映射。

非范围：
- 其他接口响应字段一致性改造（如需，另起任务）。
- 训练总结生成逻辑与数据模型变更。

## 3. 响应字段规范
- 响应数组元素字段：
  - `summary_id`
  - `user_id`
  - `source_type`
  - `source_id`
  - `log_id`
  - `completion_rate`
  - `intensity_match`
  - `recovery_advice`
  - `anomaly_notes`
  - `performance_notes`
  - `next_suggestion`
  - `deleted_at`
  - `created_at`
  - `updated_at`
- 时间字段统一为 `RFC3339`（Go `time.RFC3339`）。

## 4. 实现方式
- 在 HTTP 层新增响应 DTO 结构体，使用 JSON tag 显式映射字段名。
- 将 `[]storage.TrainingSummary` 映射为 `[]trainingSummaryResponse` 后输出。

## 5. 测试策略
- Server 测试校验返回 JSON 字段名包含 `summary_id/source_type/source_id`。
- 保持原有参数校验逻辑不变。

## 6. 风险与回滚
- 风险：前端若依赖旧字段名将产生解析失败。
- 回滚：恢复原始结构体直返即可。
