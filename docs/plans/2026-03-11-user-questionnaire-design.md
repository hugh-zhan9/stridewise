# 新用户问卷与保守模板设计 v1.0

> **版本记录**
> - v1.0 (2026-03-11)：首版，问卷字段落入 user_profiles，支持保守模板输出。

## 1. 目标与范围

**目标**
- 新用户无历史训练数据时，可通过问卷补齐基础画像并生成保守建议。
- 问卷字段落入 `user_profiles`（用户仅一份问卷）。

**非目标**
- 不做问卷历史版本管理。
- 不引入复杂个性化学习。

## 2. 问卷字段（最小闭环）

字段落在 `user_profiles`：
- `running_years`（0 / <1 / 1-3 / 3+）
- `weekly_sessions`（0-1 / 2-3 / 4+）
- `weekly_distance_km`（0-5 / 5-15 / 15-30 / 30+）
- `longest_run_km`（0 / 3 / 5 / 10 / 21）
- `recent_discomfort`（yes/no）

已存在字段复用：
- `goal_type`
- `fitness_level`

## 3. 保守模板策略

触发条件：
- 训练数据不足（`baseline.status = insufficient_data` 或 7 天训练 < 3 次）

输出策略（替代 AI）：
- `should_run`：true（若 recent_discomfort=yes 则 false）
- `workout_type`：easy_run
- `intensity_range`：低强度
- `target_volume`：根据 `weekly_distance_km` 下限给出 20%-30% 的当日建议
- `suggested_time_window`：any
- `risk_level`：green（若 recent_discomfort=yes → red）
- `explanation`：包含“问卷默认保守模板”说明

## 4. 接口与数据流

- `POST /internal/v1/user_profile`：新增字段写入。
- 推荐生成时若数据不足，走保守模板分支。

## 5. 测试策略

- 用户档案存储测试：新增字段写入/读取。
- 推荐逻辑测试：无训练数据 + 问卷字段时返回保守模板。

