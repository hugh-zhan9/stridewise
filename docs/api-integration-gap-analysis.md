# 前后端 API 字段级差异与对接分析

更新时间：2026-03-23
依据：
- 后端代码现状：`docs/2026-03-23-backend-internal-api-inventory.md`
- 前端调用需求：`docs/frontend-backend-api-requirements.md`

本文档精细到字段级别，梳理了目前前端代码期望的结构与后端 Go 代码实际实现的结构之间的差异。

## 1. 推荐与建议 (Recommendation)

### 1.1 `GET /internal/v1/recommendations/latest`

#### 字段差异：
**1. 返回结构层级不一致 (严重)**
- **后端实际返回**：包含扁平的 `input_json` (入参记录)、`output_json` (AI原始输出)、`override_json` (规则引擎覆盖)。并没有直接吐出 `workout_type`, `explanation` 等业务字段。
  ```json
  {
    "rec_id": "...", "user_id": "...", "recommendation_date": "...",
    "output_json": { /* 包含 should_run, workout_type, explanation 等 */ }
  }
  ```
- **前端期望**：直接在 `data` 下铺平所有业务字段。
  ```json
  {
    "id": "...", // 注意：前端叫 id，后端叫 rec_id
    "workout_type": "...",
    "explanation": []
  }
  ```

**2. 字段命名差异 (中等)**
- 后端主键名为 `rec_id`，前端期望名为 `id`。

#### 修复建议：
- **方案 A (推后端)**: 后端在 `formatRecommendationResponse` 时，将 `output_json` 和 `override_json` 里的字段（如 `workout_type`, `explanation` 等）拍平提取出来放在顶层，并将 `rec_id` 改为 `id`。
- **方案 B (改前端)**: 前端在 axios 的 response 拦截器里，解包 `data.output_json`，并做 `id: data.rec_id` 的映射。

### 1.2 `POST /internal/v1/recommendations/:id/feedback`

#### 字段差异：
- **请求体 (Request Body)**: 
  - 后端需要：`user_id`, `useful` (字符串), `reason` (可选)。
  - 前端实际：`useful` 传 `'useful' | 'neutral' | 'not_useful'`。
  - 后端服务校验语义值为 `yes | neutral | no`，并非直接兼容 `'useful' | 'not_useful'`，需要映射。
- **响应体 (Response Body)**:
  - 后端返回：`{ "rec_id": "...", "job_id": "..." }`
  - 前端期望：不强依赖返回值，此项 OK。

#### 修复建议：
- 前端请求前做映射：`useful -> yes`，`neutral -> neutral`，`not_useful -> no`。

---

## 2. 训练记录与基线 (Training Logs & Baseline)

### 2.1 `GET /internal/v1/baseline/current`

#### 字段差异：
**1. 结构完全不匹配 (严重)**
- **后端实际返回 `storage.BaselineCurrent`**: 
  - 包含大量底层计算数据：`data_sessions_7d`, `acute_load_srpe`, `acwr_srpe`, `monotony`, `strain`。
  - 配速相关字段为：`pace_avg_sec_per_km`, `pace_low_sec_per_km`, `pace_high_sec_per_km`（全部是秒数数字）。
- **前端期望 `BaselineResponse`**:
  - 需要高度聚合的展示数据：
  ```json
  {
    "pace_zone": { "easy": "6'00\" - 6'30\"", "tempo": "...", "interval": "..." },
    "weekly_volume_range": { "min_km": 25, "max_km": 35 },
    "recovery_level": "high" | "medium" | "low"
  }
  ```
- **核心缺失**：后端目前只返回了单点配速秒数，没有直接提供格式化好的配速区间字符串，也**没有提供建议的周跑量 `weekly_volume_range`**。`recovery_level` 在后端可能叫 `status` 或者是 `recovery_scores` 表里的数据。

#### 修复建议：
- **必需改后端**：前端无法直接从 ACWR 和 Strain 中算出友好的 `pace_zone` 和 `weekly_volume_range`。后端需要在返回 `BaselineCurrent` 时增加聚合转化逻辑。

### 2.2 `GET /internal/v1/training/logs`

#### 字段差异：
**1. 字段映射需补齐 (中等)**
- **前端已做了部分映射**：在 `api/training.ts` 中前端已经将后端的 `log_id` 映射为 `id`，`duration_sec` 映射为 `duration_min`，`pace_str` 映射为 `avg_pace`。
- **潜在隐患**：前端需要 `train_date_local`（本地时间日期格式用于展示），但后端返回的是 `start_time` (RFC3339 时间戳)。前端需要确保能正确解析时区。

### 2.3 `POST /internal/v1/training/logs` (新增手动记录)

#### 字段差异：
**1. 枚举值严重不匹配 (严重)**
- **前端传参 `training_type`**: `'easy_run' | 'long_run' | 'interval' | 'strength' | 'rest'` 等。
- **后端期望 `training_type`**: 后端中文枚举 `'轻松跑' | '有氧跑' | '间歇跑' | '长距离'`。如果传英文，后端会将其全部归类为 `'custom'` 并在 `training_type_custom` 里记录。
- **时间与配速格式**: 
  - 前端：`start_time: "2026-03-12 00:00:00"`, `duration: "30m"`, `pace: "06:00"` (冒号)。
  - 后端：`pace` 强校验为 `mm'ss''` 格式（如 `06'00''`）。如果收到带冒号的可能会报错。

#### 修复建议：
- **改前端**：前端在提交前，必须将配速里的 `:` 替换为 `'{分}''{秒}''` 格式，并且将 `training_type` 的英文字段映射为后端的中文枚举。

---

## 3. 用户档案 (Profile)

### 3.1 `POST /internal/v1/user/profile` (创建/更新档案)

#### 字段差异：
**1. 必填项缺失导致必定 400 Bad Request (致命)**
- **前端当前调用**：在“调整参数”弹窗里，前端只传递了 `user_id`, `goal_type`, `goal_target`。
- **后端实际强校验**：后端代码中存在 `validateUserProfileRequest` 方法，强制要求：
  - `gender`
  - `age` (>0)
  - `height_cm` (>0)
  - `weight_kg` (>0)
  - `running_years` (必须是 `0/<1/1-3/3+` 之一)
  - `weekly_sessions`
  - `weekly_distance_km`
  - `longest_run_km`
  - `recent_discomfort`
  - `location_lat` (-90~90)
  - `location_lng` (-180~180)
  - `country`, `province`, `city`
- **结果**：前端目前的增量提交会直接被后端打回。

#### 修复建议：
- **最佳方案 (后端)**: 后端提供一个专用的局部更新接口（如 `PUT /internal/v1/user/profile/goal`，这个在早期的 OpenAPI 里有，但现状代码里没有实现），仅校验目标字段。
- **备选方案 (前端)**: 前端在提交前，先 `GET` 全量数据，然后 `Merge` 更新目标字段，最后将这个庞大的对象原封不动 `POST` 回去。

### 3.2 `GET /internal/v1/user/profile`

#### 字段差异：
**1. 返回枚举对不齐 (中等)**
- **前端期望 `ability_level`**: `'beginner' | 'intermediate' | 'advanced'`。
- **后端实际返回**：后端当前似乎没有直接返回 `ability_level_reason` 和 `ability_level_updated_at`，或者格式有出入。而且如果是新创建的用户，后端的 `fitness_level` 写死为 `"unknown"`，并未返回具体的中文等级。

---

## 4. 全局总结与行动项

目前前后端处于**“大体结构一致，但细微校验和枚举完全对不齐”**的阶段。

### 4.1 未覆盖的接口差异（本次补充）

以下接口已在后端清单中存在，但在前端需求文档中尚未纳入，也未在本分析文档旧版本中体现：

1. `POST /internal/v1/sync/jobs`
2. `GET /internal/v1/sync/jobs/{job_id}`
3. `POST /internal/v1/sync/jobs/{job_id}/retry`
4. `POST /internal/v1/weather/snapshot`
5. `GET /internal/v1/weather/risk`
6. `GET /internal/v1/trends`
7. `GET /internal/v1/training/summaries`
8. `POST /internal/v1/training/feedback`
9. `GET /internal/health`
10. `GET /internal/metrics`

说明：
- 这些接口是否“缺失”取决于前端当前版本是否要消费。若前端暂不使用，可标记为“后端已提供，前端未接入（非阻断）”。
- 若进入联调范围，需补充到 `docs/frontend-backend-api-requirements.md`，并在本文件增加字段级映射。

**必须立即解决的阻断性问题（Blockers）**：
1. **档案更新报错**：修改后端的 `/profile` 接口支持局部更新，或在前端补齐所有冗长且必填的体征与经纬度假数据。
2. **配速格式报错**：前端新增训练记录时，`pace` 必须传类似 `05'30''` 的格式。
3. **主页数据为空**：前端需要将后端返回的 `latest` 推荐数据中的 `output_json` 拍平，否则前端拿不到 `workout_type` 导致白屏。
4. **基线能力数据缺漏**：后端 `BaselineCurrent` 接口必须补充返回前端需要的配速区间字符串和建议跑量范围。

**非阻断但应跟踪的问题**：
1. **反馈语义值不一致**：`useful/neutral/not_useful` 与 `yes/neutral/no` 需统一映射策略并文档化。
2. **接口范围不一致**：后端已提供同步/天气/趋势/训练反馈能力，但前端需求文档未纳入，后续会造成“接口存在但无人使用”的漂移。
