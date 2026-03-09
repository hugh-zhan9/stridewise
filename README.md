# StrideWise / 睿步

AI 驱动的智能跑步训练助手。

StrideWise（睿步）通过天气、训练历史、能力基线与训练反馈，为不同水平跑者生成每日个性化训练建议，并通过安全规则与反馈闭环持续优化建议质量。

## 产品定位
- 中文名：睿步
- 英文名：StrideWise
- 产品类型：AI 运动建议产品（跑步场景）
- 核心理念：AI 个性化分析 + 规则安全约束 + 训练反馈闭环

## 核心能力
- 用户建档与目标设置（新手/进阶可区分）
- 能力基线评估与动态更新
- 天气风险分级（绿/黄/红）
- 每日训练建议（可解释）
- 训练后总结与“建议是否有用”反馈
- 异常兜底与安全降级（AI 失败可回退）

## 业务闭环
1. 用户设置目标并完成基础建档
2. 系统聚合天气、近 7 天训练负荷、能力基线
3. 生成当日建议（AI 主路径，规则兜底）
4. 用户执行训练并记录/导入数据
5. 系统输出训练总结并收集反馈
6. 系统更新能力基线，驱动下一次建议

## 技术方案概览
- 架构形态：Hybrid Decision Engine
- 决策方式：
  - AI Advisor：生成个性化候选建议
  - Rule Engine：安全门禁、冲突裁决、fallback 兜底
- 数据能力：建议、训练、基线、反馈全链路可追溯
- 数据采集策略：P0 首发 Keep，后续扩展 Strava/Garmin/Nike/GPX/TCX/FIT
- 合规原则：非医疗建议、最小权限、可撤销授权

## 项目阶段
当前仓库处于“需求与详细设计阶段”，已完成：
- 需求文档（request）
- 详细设计 V2
- OpenAPI + AI 输入输出 JSON Schema
- 文档版本管理规范

## 文档导航
- 需求文档：`request.md`
- 详细设计（主文档）：`docs/plans/2026-03-09-stridewise-detailed-design-v2.md`
- 详细设计附录（OpenAPI + Schema）：`docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`
- 开发计划：`docs/plans/2026-03-09-stridewise-development-plan-v1.md`
- 数据采集与标准化设计：`docs/plans/2026-03-09-stridewise-data-ingestion-design-v1.md`
- 技术选型总结：`docs/plans/2026-03-09-stridewise-tech-selection-summary-v1.md`
- 文档版本规范：`docs/plans/DOC-VERSIONING.md`

## MVP 目标指标
- 建议采纳率 >= 35%
- 建议后训练完成率 >= 70%
- 7日留存率 >= 25%
- 训练后不适反馈率 <= 5%

## 近期里程碑
- M1：基础数据链路打通
- M2：训练建议闭环打通
- M3：内测上线

## License
待定
