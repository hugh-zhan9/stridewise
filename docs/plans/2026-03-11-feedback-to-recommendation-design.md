# 训练反馈影响下一次建议设计 v1.0.0

> **版本记录**
> - v1.0.0 (2026-03-11)：首版，训练反馈与总结进入 AI 输入，用于下一次建议个性化。

## 1. 目标与范围

**目标**
- 在生成每日训练建议时读取最近一条训练反馈（含自由文本）。
- 若存在自由文本，则将“自由文本 + 对应训练总结结构化字段”写入 AI 输入。
- 让 AI 在建议中参考上一条训练反馈，给出更合适的建议。

**范围**
- 数据来源包含手动训练记录（log）与第三方活动（activity）。
- 仅处理最近一条有自由文本的训练反馈。
- 不改变训练反馈/训练总结存储结构。

**非目标**
- 不引入额外缓存或异步洞察。
- 不在无自由文本时传递结构化总结。

## 2. 数据结构扩展

在 AI 推荐输入新增字段：

```json
"latest_training_feedback": {
  "source_type": "log|activity",
  "source_id": "xxx",
  "created_at": "RFC3339",
  "content": "用户自由文本",
  "summary": {
    "completion_rate": "...",
    "intensity_match": "...",
    "recovery_advice": "...",
    "anomaly_notes": "...",
    "performance_notes": "...",
    "next_suggestion": "..."
  }
}
```

规则：
- 仅当最近反馈 `content` 非空时输出该字段。
- `summary` 与该反馈对应的训练总结（同 `source_type/source_id`）。

## 3. 数据流与实现位置

- 在 `recommendation.Processor.Generate` 中：
  1. 查询最近一条训练反馈（log/activity），要求 `content` 非空。
  2. 查询该反馈对应的训练总结。
  3. 组装 `latest_training_feedback` 写入 AI 输入。

## 4. 测试策略

- 单测：存在自由文本时 `latest_training_feedback` 写入 AI 输入。
- 单测：无自由文本时不写入该字段。

## 5. 风险与回滚

- 风险：自由文本可能影响模型输出。
- 缓解：规则兜底仍适用，且结构化总结同时输入。
- 回滚：移除 AI 输入字段即可恢复原行为。
