# 训练建议解释数量软约束设计

## 文档版本
- 当前版本：v0.1.0
- 发布日期：2026-03-11
- 文档状态：草稿

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v0.1.0 | 2026-03-11 | 新增解释数量软约束设计，移除硬校验与 Schema 最小条数限制 |

## 背景
当前 AI 输出的 `explanation` 被硬性要求至少 2 条，且在 OpenAI 结果校验中直接触发失败回退。这与“允许少于两条也直接返回 AI 结果”的新策略冲突，导致不必要的回退与结果丢失。

## 目标
- `explanation` 仅做“期望 ≥2 条”的软约束，不再作为失败条件。
- AI 输出少于 2 条时仍返回 AI 结果。
- OpenAPI/JSON Schema 移除 `minItems` 约束。

## 非目标
- 不引入新的兜底补齐逻辑。
- 不新增监控或审计字段（如需统计，可后续独立设计）。

## 行为变更
1. AI 输出校验：移除 `explanation` 最小条数校验，只校验关键字段非空。
2. Prompt：继续强调“至少两条解释”，但不做硬性校验。
3. Schema：输出字段 `explanation` 不再限制 `minItems`。

## 影响范围
### 代码
- `backend/internal/ai/openai_recommender.go`
  - `validateRecommendationOutput` 移除 `explanation` 最小条数校验。
- `backend/internal/recommendation/processor_test.go`
  - 调整测试：不再依赖“解释≥2”，仅校验存在或非空（按现有断言需要更新）。

### 文档
- `request.md`
  - 将“解释≥2”改为“期望≥2，允许不足”。
- `docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`
  - 删除 `explanation` 的 `minItems` 约束。

## 测试策略
- 新增/调整单测覆盖：AI 输出解释条数不足时仍视为有效输出。
- 运行推荐模块相关测试用例，确保回退行为不被触发。

## 风险与回滚
- 风险：AI 解释条数不足的体验下降，但功能不回退。
- 回滚：恢复 `explanation` 最小条数校验与 Schema 约束。
