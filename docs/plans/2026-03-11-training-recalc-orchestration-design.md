# 训练记录变更异步编排与手动限制设计

## 文档版本
- 当前版本：v0.1.0
- 发布日期：2026-03-11
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v0.1.0 | 2026-03-11 | 新增训练记录更新/删除的异步编排与 manual 限制设计。 |

## 背景与问题
当前训练记录更新/删除后仅入队 `baseline:recalc`，且 `training:recalc` 处理器为空实现，导致“刷新建议、回滚训练总结/反馈”等 P0 要求无法完成。同时，接口未限制只允许 `manual` 记录更新/删除，存在误操作风险。

## 目标
- 训练记录新增/更新/删除后统一入队 `training:recalc` 编排任务。
- 仅允许 `manual` 来源记录更新/删除；第三方记录禁止修改删除。
- 更新/删除触发：基线重算、建议刷新、总结/反馈回滚（删除时）。

## 非目标
- 不引入新的队列或多节点调度。
- 不改变已有训练记录冲突检测规则。

## 方案
### 1. API 行为调整
- `POST/PUT/DELETE /internal/v1/training/logs`：入队 `training:recalc`（替代 `baseline:recalc`）。
- 更新/删除前必须读取 `training_logs`，校验 `source == manual`。
  - 非 manual 直接返回 400（或 403，当前统一 400）。

### 2. 训练重算编排（training:recalc）
`training.Processor` 承担编排职责：
1. **recalcBaseline**：触发基线重算（调用 baseline 处理能力）。
2. **refreshRecommendation**：直接调用 `recommendation.Generate` 生成新建议。
3. **rollbackSummaryAndFeedback**：
   - `delete`：软删 `training_summaries` 和 `training_feedbacks`（按 `source_type=log` / `source_id=log_id`）。
   - `update`：重新生成 summary（复用已有 AI 总结流程）。

失败处理：任何步骤失败 → `async_jobs` 标记 `failed` 并写入错误信息；成功 → `success`。

### 3. 依赖注入调整
`training.Processor` 需新增依赖：
- Baseline 处理器（或其核心接口）
- Recommendation 服务
- TrainingSummary/TrainingFeedback 读写能力

## 数据与接口影响
- 无新增表结构。
- 仅改变异步任务类型与处理流程。

## 测试策略
- 单测覆盖：
  - `training:recalc` 在 `update/delete` 时触发 summary/feedback 回滚。
  - 非 manual 更新/删除返回 400。
- 集成测试：
  - 训练记录更新/删除后，`async_jobs` 状态与新建议生成链路可达。

## 风险与回滚
- 风险：训练重算中建议刷新失败会导致建议延迟更新。
- 回滚：恢复为仅入队 `baseline:recalc`，并关闭 `training:recalc` 处理。
