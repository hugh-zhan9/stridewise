# 恢复状态规则设计 v1.0

> **版本记录**
> - v1.0 (2026-03-11)：首版，基于 ACWR/Monotony 阈值推断恢复状态并进入规则兜底。

## 1. 目标与范围

**目标**
- 在推荐生成时计算恢复状态（green/yellow/red）。
- 触发安全降级时必须优先兜底（与天气/不适/高负荷并列）。

**非目标**
- 不引入外部睡眠/心率数据。
- 不实现“周环比增量（10%规则）”。

## 2. 指标与依据（保守阈值，不作医学结论）

- ACWR 常被引用的“低风险区间”为 0.8–1.3，≥1.5 风险上升。该指标存在方法学争议，仅用于保守安全阈值。([efsma.org](https://efsma.org/images/pdf/publications/Br-J-Sports-Med-2016-Gabbett-273-80.pdf))
- “10% 规则”缺乏证据支撑，因此不采用。([pmc.ncbi.nlm.nih.gov](https://pmc.ncbi.nlm.nih.gov/articles/PMC6253751/))
- Monotony/Strain 采用 Foster 定义（均值/标准差与周负荷×单调性），Monotony > 2.0 常被视作风险提示。([jssm.org](https://www.jssm.org/volume03/iss4/cap/jssm-03-234.pdf))

## 3. 判定规则

计算：
- `ACWR = max(ACWRSRPE, ACWRDistance)`

阈值映射：
- **Green**：ACWR ∈ [0.8, 1.3] 且 Monotony < 2.0
- **Yellow**：ACWR ∈ (1.3, 1.5] 或 Monotony ∈ [2.0, 2.2)
- **Red**：ACWR > 1.5 或 Monotony ≥ 2.2（ACWR ≥ 2.0 直接 Red）

输出策略：
- 任何 Red → `recovery_status=red`
- 否则任何 Yellow → `recovery_status=yellow`
- 否则 `recovery_status=green`

## 4. 落地位置与数据流

- 在 `recommendation.Processor.Generate` 内计算恢复状态。
- 通过规则输入（`RuleInput`）进入 `ApplyRules`。
- 恢复状态优先级：**安全 > 恢复 > 天气 > 高负荷 > 目标**（恢复不足触发与安全规则等价的降级输出）。

## 5. 测试策略

- 单测：恢复状态计算边界值
- 规则单测：恢复不足触发降级与替代方案
- 集成测试：推荐输出包含恢复状态与解释

