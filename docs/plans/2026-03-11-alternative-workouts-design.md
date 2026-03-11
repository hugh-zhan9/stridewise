# 安全降级触发替代运动建议设计 v1.0

> **版本记录**
> - v1.0 (2026-03-11)：首版，安全降级时补充替代运动建议输出结构。

## 1. 目标与范围

**目标**
- 当触发任何安全降级（天气红色 / 不适 / 高负荷）时，必须返回可执行的替代运动建议。
- 保持“AI 优先 + 规则兜底”，规则保证替代建议存在。

**非目标**
- 不改变现有 AI 生成主体建议结构。
- 不引入新的外部接口。

## 2. 数据结构扩展

在 `RecommendationOutput` 增加 `alternative_workouts` 字段：

```json
"alternative_workouts": [
  {"type":"treadmill","title":"室内跑步机轻松跑","duration_min":30,"intensity":"low"},
  {"type":"strength","title":"基础力量训练","duration_min":20,"intensity":"low"},
  {"type":"mobility","title":"拉伸与灵活性恢复","duration_min":15,"intensity":"low"}
]
```

字段定义：
- `type`: `treadmill` / `strength` / `mobility` / `rest`
- `title`: 简短标题
- `duration_min`: 建议时长（可选）
- `intensity`: `low` / `medium`
- `tips`: 可选注意事项

## 3. 规则兜底逻辑

触发条件：任意安全降级（`weather_red` / `user_discomfort` / `high_load`）。

规则行为：
- `should_run=false`
- 若 `alternative_workouts` 为空，填入默认替代方案（跑步机/力量/拉伸）
- 若解释不足 2 条，补齐安全原因解释

## 4. AI 输出与校验

- AI 可选输出 `alternative_workouts`，不强制校验。
- 若 AI 未输出，规则兜底生成。

## 5. 测试策略

- 规则单测：安全降级时 `alternative_workouts` 必须存在。
- 推荐流程单测：输出 JSON 中包含替代建议。
- `convertOutput` 映射单测：AI 输出的替代方案可正确持久化。

