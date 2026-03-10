# StrideWise 每日训练建议引擎设计文档

## 文档版本
- 当前版本：v0.1.0
- 发布日期：2026-03-10
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v0.1.0 | 2026-03-10 | 初版：点击生成、AI 优先 + 规则兜底、数据模型与异常策略。 |

## 1. 背景与目标
P0 的核心链路是“用户点击获取建议 → 生成结构化建议 → 安全规则兜底 → 落库返回”。
目标：保证建议可解释、可追溯、可降级，且生成延迟可控。

## 2. 核心流程（点击即生成）
1. 用户点击“获取建议”
2. 拉取最新天气并保存快照
3. 读取用户档案、能力基线、近 7 天训练负荷、最近不适反馈
4. 构造 AI 输入（包含红色风险/恢复不足/不适等信号）
5. 调用 OpenAI 生成结构化建议
6. 规则兜底校验（可覆盖 AI 输出）
7. 保存 Recommendation 记录（含输入/输出摘要）
8. 返回建议给前端

说明：同一天多次点击不复用，按时间因素重新生成并落库多条记录。

## 3. 数据模型

### 3.1 Recommendation
表：`recommendations`
- `rec_id`
- `user_id`
- `created_at`
- `recommendation_date`（日期）
- `input_json`（AI 输入摘要）
- `output_json`（AI 输出）
- `risk_level`
- `override_json`（规则覆盖详情）
- `is_fallback`（AI 失败是否降级）
- `ai_provider`
- `ai_model`
- `prompt_version`
- `engine_version`

### 3.2 RecommendationFeedback
表：`recommendation_feedbacks`
- `feedback_id`
- `rec_id`
- `user_id`
- `useful`（`yes/neutral/no`）
- `reason`（可选）
- `created_at`

## 4. 规则兜底（硬约束）
- 天气红色风险：强制休息/替代方案
- 近期高负荷且恢复不足：强制降级为低强度或休息
- 用户上报不适：强制休息

规则覆盖需记录 `override_reason`，写入 `override_json`。

## 5. 异常与兜底
- AI 超时/失败：写 `is_fallback=true`，输出规则保守建议
- 天气失败：使用最近一次快照并标记异常；若无快照，输出保守建议
- 基线缺失：使用低风险模板

## 6. 接口（内部）
- `POST /internal/v1/recommendations/generate`
- `GET /internal/v1/recommendations/latest?user_id=...`
- `POST /internal/v1/recommendations/{id}/feedback`

## 7. 测试策略
- AI 调用失败回退
- 规则覆盖场景
- 多次点击生成多条记录
- 反馈幂等（同一建议仅允许一次有效反馈）
