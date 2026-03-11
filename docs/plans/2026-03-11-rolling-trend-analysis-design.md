# StrideWise 滚动训练趋势分析设计

## 文档版本
- 当前版本：v0.1.0
- 发布日期：2026-03-11
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v0.1.0 | 2026-03-11 | 新增滚动 7/30 天训练趋势分析设计（P1-3）。 |

## 1. 背景与目标
P1 需求要求提供周/月训练趋势分析。为避免自然周/月口径不一致，本次采用**滚动 7 天 / 30 天**口径，统一用于趋势展示与指标计算。

目标：
- 输出滚动窗口内的训练趋势（按天序列）。
- 提供可解释的汇总指标与负荷趋势快照。
- 不依赖多数据源扩展即可落地。

## 2. 范围与非目标
范围内：
- 滚动 7/30 天趋势分析 API（内部）。
- 基础指标 + 扩展指标 + 负荷指标。

非目标：
- 自然周/月统计口径。
- 复杂模型化指标（如 VO2max 估计）。
- 缓存/预计算数据表（先走实时聚合）。

## 3. 统计口径
- `window=7d`：以 `as_of` 向前滚动 7 天（含 `as_of` 当天）。
- `window=30d`：以 `as_of` 向前滚动 30 天（含 `as_of` 当天）。
- `as_of` 为空时默认当前时间。

## 4. 指标定义
### 4.1 基础指标（summary）
- `sessions`：窗口内训练次数（手动记录 + 第三方活动）。
- `distance_km`：总里程（km）。
- `duration_sec`：总时长（秒）。
- `avg_pace_sec_per_km`：平均配速（按里程加权）。
- `avg_rpe`：平均 RPE（仅手动记录且 rpe>0）。

### 4.2 扩展指标（summary）
- `summary_count`：训练总结记录数。
- `completion_rate_dist`：完成度分布（low/mid/high/unknown）。
- `intensity_match_dist`：强度匹配分布（low/mid/high/unknown）。
- `recovery_advice_tags`：恢复建议关键词计数（轻量规则提取）。

### 4.3 负荷指标（summary）
- `acwr_srpe`、`acwr_distance`、`monotony`、`strain`：取 `baseline_history` 中 `as_of` 最近一条。
- 若无基线历史则返回 `null`。

### 4.4 趋势序列（series）
- 以天为粒度的序列点（`date`），用于折线图。
- 每日指标：`distance_km`、`duration_sec`、`avg_pace_sec_per_km`、`avg_rpe`、`sessions`。

## 5. 数据来源
- `training_logs`：手动训练记录。
- `activities`：第三方活动记录。
- `training_summaries`：训练总结（结构化字段）。
- `baseline_history`：基线历史快照（负荷趋势）。

## 6. 聚合规则
- **平均配速**：按里程加权平均：`sum(pace_sec_per_km * distance_km) / sum(distance_km)`。
- **平均 RPE**：仅纳入 `training_logs` 且 `rpe>0`。
- **完成度/强度匹配映射**：基于关键词映射到 low/mid/high，无法识别计入 unknown。

关键词映射示例（可配置/可扩展）：
- low：`低`、`不足`、`偏低`
- mid：`适中`、`匹配`、`正常`
- high：`高`、`过强`、`偏高`

## 7. API 设计（内部）
- `GET /internal/v1/trends`
- 参数：
  - `user_id`（必填）
  - `window`（必填，`7d`/`30d`）
  - `as_of`（可选，`YYYY-MM-DD` 或 `YYYY-MM-DD HH:MM:SS`）
- 响应：Envelope 包裹，`data` 为趋势对象。

## 8. 异常处理
- 窗口无数据：`summary` 为 0/空集合，`series` 为空数组。
- `as_of` 非法：返回 400。
- `window` 非法：返回 400。

## 9. 性能与约束
- 当前仅支持滚动 7/30 天窗口，数据量可控。
- 暂不做缓存与预计算，后续可扩展。

## 10. 测试策略
- 单测覆盖：聚合口径、加权配速、RPE 平均、关键词映射。
- 接口测试：参数校验与空窗口返回。
