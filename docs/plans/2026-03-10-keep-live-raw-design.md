# StrideWise Keep 轨迹/心率 Raw 写入设计 V1

> 目标：在 Keep 逆向登录采集的基础上，完整写入 `geoPoints` 与 `heartRates` 的原文与解码结果，保持 running_page 行为一致。

## 文档版本
- 当前版本：v1.0.0
- 发布日期：2026-03-10
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v1.0.0 | 2026-03-10 | 首次发布 Keep 轨迹/心率 Raw 写入设计。 |

## 1. 设计约束
1. 解析逻辑严格对齐 `running_page/run_page/keep_sync.py`。
2. Raw 需保留原文 + 解码结果，不做压缩与裁剪。
3. 失败单条记录跳过，不阻断任务。

## 2. Raw 写入策略
新增 Raw 字段：
- `geo_points_raw`：Keep 返回的原始 `geoPoints` 字符串
- `heart_rate_raw`：Keep 返回的原始 `heartRate` 对象
- `geo_points_decoded`：解码后的轨迹点数组
- `heart_rates_decoded`：解码后的心率点数组

## 3. 解码逻辑（running_page 对齐）
1. `base64 -> AES-CBC -> zlib/gzip -> JSON`
2. `TRANS_GCJ02_TO_WGS84 = true`（轨迹点坐标转换）
3. 心率点不额外处理，仅解码后写入 Raw

## 4. 性能与体积
1. Raw JSON 直接写入 `raw_activities.payload_json`
2. 不额外压缩与裁剪
3. 若解码失败则跳过该条记录

## 5. 验收标准
1. Raw 同时包含原文与解码结果
2. `summary_polyline` 与轨迹点一致
3. 解码失败不影响整体任务完成
