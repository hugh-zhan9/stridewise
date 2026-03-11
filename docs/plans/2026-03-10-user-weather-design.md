# StrideWise 用户档案与天气模块设计文档

## 文档版本
- ~~当前版本：v0.1.0~~
- ~~当前版本：v0.2.0~~
- ~~当前版本：v0.3.0~~
- ~~当前版本：v0.4.0~~
- 当前版本：v0.5.0
- 发布日期：2026-03-11
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v0.5.0 | 2026-03-11 | 能力层级改为 AI 自动判定，移除手动输入字段。 |
| v0.4.0 | 2026-03-10 | ~~包含风寒/低温风险阈值~~ 移除冷风险（P0 不纳入），并调整降水仅触发黄色提示；引用最新来源修订参考说明。 |
| v0.3.0 | 2026-03-10 | ~~P0 风险分级未给出数值阈值~~ 增补专业参考阈值（AQI/UV/热指数/风速/风寒）并注明降水概率为工程阈值。 |
| v0.2.0 | 2026-03-10 | ~~仅保留风险分级/接口/验收片段~~ 补齐目标/范围/数据模型/流程/异常/测试，并对齐“AI 优先 + 规则兜底”和“位置必填”。 |
| v0.1.0 | 2026-03-10 | 初版（内容不完整）。 |

## 1. 背景与目标
用户档案与天气模块为建议引擎提供必要的输入。当前策略为“AI 优先 + 规则兜底”：AI 负责生成建议，规则负责安全校验与强制降级。

目标：
- 确保用户位置必填且可信，为天气风险与建议生成提供稳定输入。
- 天气数据标准化与风险分级，为规则兜底提供明确阈值。
- 为后续接入真实天气源保留 Adapter 结构。

## 2. 范围与边界
范围内：
- 用户档案创建/更新（含位置必填与来源记录）
- 天气快照采集（Mock Provider）与风险评估
- 内部接口：档案/快照/风险查询

范围外：
- AI 建议生成逻辑本体（由建议服务负责）
- 第三方用户鉴权（当前阶段无完整鉴权）

## 3. 设计原则
- AI 优先：AI 先生成建议，再由规则兜底校验。
- 安全优先：红色风险强制降级或替代方案。
- 位置必填：缺失位置不得生成天气快照与建议。
- 内部接口：所有接口均为内部调用，不对外开放。

## 4. 数据模型（MVP）

### 4.1 UserProfile
字段（位置必填）：
- user_id
- gender, age, height, weight
- goal_type, goal_cycle, goal_frequency, goal_pace
- ~~fitness_level~~
- ability_level（AI 自动判定）
- location_lat, location_lng
- country, province, city
- location_source（geo/manual）
- created_at, updated_at

校验规则：
- 位置字段缺失返回 400
- 位置来源必须为 geo/manual

### 4.2 WeatherSnapshot
字段：
- snapshot_id, user_id, date
- temperature, feels_like, humidity, wind_speed
- precipitation_prob, aqi, uv_index
- risk_level（green/yellow/red）
- created_at

## 5. 核心流程

### 5.1 用户档案创建/更新
1. 前端提供定位（浏览器/APP）或用户手动调整。
2. 服务端校验位置必填。
3. 保存 UserProfile。

### 5.2 天气快照生成
1. 根据 user_id 获取位置。
2. 调用 Weather Provider（当前为 Mock）获取天气字段。
3. 风险分级计算并写入 WeatherSnapshot。

### 5.3 风险分级（P0 规则版）
~~- **红色**：任一条件触发：极端温度 / 高风 / 高降水 / 高 AQI / 高 UV~~
~~- **黄色**：中等风险阈值触发（降强度/缩时长）~~
~~- **绿色**：其余~~
~~天气缺失：输出保守建议并标记异常。~~

**规则说明：** 任一指标触发红色即红色；否则若任一指标触发黄色即黄色；其余为绿色。

**阈值（专业参考 + 工程兜底）：**
- **AQI（空气质量）**
  - 黄色：101-150（对敏感人群不健康）
  - 红色：>=151（不健康及以上）
- **UV 指数**
  - 黄色：3-7（中等到高）
  - 红色：>=8（很高到极端）
- **热指数/体感温度（以 `feels_like` 为准）**
  - 黄色：90-105°F（32-40.6°C）
  - 红色：>=105°F（>=40.6°C）
- ~~**风寒/体感温度（以 `feels_like` 为准）**~~
  - ~~黄色：-15°F 到 -24°F（-26°C 到 -31°C）~~
  - ~~红色：<= -25°F（<= -32°C）~~
- 冷风险：P0 暂不纳入（场景假设低温条件下不进行户外跑步），后续可扩展。
- **风速（持续风，`wind_speed_ms`）**
  - 黄色：31-39 mph（14-17 m/s）
  - 红色：>=40 mph（>=18 m/s）
- **降水概率（工程阈值，因仅有概率字段）**
  - 黄色：>=40%
  - ~~红色：>=70%~~
  - 仅触发黄色提示，不单独触发红色。

天气缺失：输出保守建议并标记异常。

**阈值参考来源：**
- ~~AQI 分类：AirNow 官方 AQI 分类标准。 citeturn0search0~~
- ~~UV 风险级别：EPA UV Index Scale。 citeturn0search1~~
- ~~热指数风险分级：NWS Heat Index 风险表（90-105/105-130 等）。 citeturn1search3~~
- ~~风寒与极端低温：NWS Cold Weather Advisory/Extreme Cold Warning 标准。 citeturn1search0~~
- ~~大风警报阈值：NWS Wind Advisory/High Wind Warning 标准。 citeturn1search7~~
- AQI 分类：AirNow AQI Basics。
- UV 风险级别：EPA UV Index Scale。
- 热指数风险分级：NWS Heat Safety（Heat Index 90-105/105-130）。
- 风速阈值：NWS Wind Advisory/High Wind Warning。

## 6. Weather Provider 设计
接口：
- `GetSnapshot(location)` -> `WeatherSnapshotInput`

实现：
- `MockProvider`：用于当前阶段联调与测试。
- 预留扩展：后续可接入真实天气源并替换实现。

## 7. 内部接口
1. `POST /internal/v1/user/profile`：创建/更新档案（含位置必填）
2. `GET /internal/v1/user/profile?user_id=...`
3. `POST /internal/v1/weather/snapshot`：内部调用生成快照
4. `GET /internal/v1/weather/risk?user_id=...&date=...`

## 8. 异常与兜底
- 位置缺失：返回 400，提示补齐位置。
- 天气拉取失败：使用最近一次快照并标记异常；若无历史快照，输出保守建议。
- 风险红色：强制降级或替代方案。

## 9. 测试计划
- 单元测试：风险阈值边界、位置必填校验
- 集成测试：档案创建 -> 快照生成 -> 风险查询
- 异常测试：天气失败/位置缺失/红色风险强制降级

## 10. 验收标准
1. 位置必填校验生效
2. 天气快照可生成并入库
3. 风险等级正确输出（green/yellow/red）
