# StrideWise 能力层级自动判定设计 v1.0

> **版本记录**
> - v1.0 (2026-03-11)：首版，能力层级完全由 AI 自动判定，失败沿用历史，无历史则阻断建议并异步重试。

## 1. 目标与范围

**目标**
- 能力层级由 AI 基于历史训练数据与问卷自动判定。
- 失败时不降级为手动输入，沿用历史能力层级；无历史则阻断建议并触发异步重试。

**非目标**
- 不支持用户手动覆盖能力层级。
- 不做模型个性化学习与长期漂移控制。

## 2. 关键原则
- AI 为唯一能力层级来源。
- 建议生成对能力层级强依赖：无值即阻断。
- 任务异步化，降低建议生成延迟。

## 3. 数据模型

在 `user_profiles` 中新增或替代字段：
- `ability_level`（`beginner|intermediate|advanced`）
- `ability_level_updated_at`（最近成功计算时间）
- `ability_level_reason`（可选，AI 输出摘要）

说明：原 `fitness_level` 字段不再作为输入字段，后续可废弃或迁移为 `ability_level`。

## 4. AI 判定输入/输出

**输入（摘要化）**
- 近 28 天训练摘要：次数、总里程、总时长、平均配速、sRPE 指标
- 问卷字段：`running_years`、`weekly_sessions`、`weekly_distance_km`、`longest_run_km`
- 基础档案：年龄、体重

**输出**
- `ability_level`（必填）
- `reason`（可选，文本说明）

## 5. 触发时机
- 新用户问卷保存成功 → 异步触发 `ability_level:calc`
- 训练记录新增/更新/删除 → 异步触发 `ability_level:calc`
- 第三方同步完成 → 异步触发 `ability_level:calc`
- AI 失败/超时 → 异步重试

## 6. 建议生成的阻断与兜底
- 若 `ability_level` 存在：正常生成建议。
- 若 `ability_level` 不存在：返回 `ability_level_not_ready` 错误，阻断建议，并创建 `ability_level:calc` 任务。
- 若 AI 失败且存在历史能力层级：沿用历史能力层级，记录失败原因并异步重试。

## 7. 任务与幂等
- 任务类型：`ability_level:calc`
- 同一用户短时间内重复触发应合并：若已有 queued/running 任务则复用。
- 任务失败不清空现有能力层级。

## 8. 测试策略
- Storage：能力层级字段读写、更新时间写入。
- Server：profile 接口不再接收 `fitness_level`；建议生成在 `ability_level` 为空时阻断。
- Worker：任务成功写回；失败不覆盖旧值。

## 9. 文档与接口同步
- `request.md` 删除“能力层级支持手动调整”的描述。
- OpenAPI 与 AI Schema 移除 `fitness_level` 输入并新增 `ability_level` 输出。

