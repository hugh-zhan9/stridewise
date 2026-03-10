P0 规则版：
- **红色**：任一条件触发：极端温度 / 高风 / 高降水 / 高 AQI / 高 UV
- **黄色**：中等风险阈值触发（降强度/缩时长）
- **绿色**：其余

天气缺失：输出保守建议并标记异常。

## 5. 接口与数据流
1. `POST /internal/v1/user/profile`：创建/更新档案（含位置必填）
2. `GET /internal/v1/user/profile?user_id=...`
3. `POST /internal/v1/weather/snapshot`：内部调用生成快照
4. `GET /internal/v1/weather/risk?user_id=...&date=...`

## 6. 验收标准
1. 位置必填校验生效
2. 天气快照可生成并入库
3. 风险等级正确输出（green/yellow/red）
