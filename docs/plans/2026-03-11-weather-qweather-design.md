# QWeather 天气接入与三日预报落库设计 v1.1

> **版本记录**
> - v1.0 (2026-03-11)：首次设计，新增 QWeather 实时天气 + 三日预报落库与 AI 输入扩展。
> - v1.1 (2026-03-11)：允许预报字段为 NULL，~~移除预报 AQI~~。

## 1. 目标与范围

**目标**
- 使用 QWeather 获取实时天气与未来 3 天预报。
- 实时天气写入 `weather_snapshots`，三日预报写入新增表 `weather_forecasts`。
- 训练建议输入包含未来 3 天窗口信息。
- 所有接口保持内部使用，不新增外部 API。

**非目标**
- 不引入多源天气聚合。
- 不改动前端交互。

## 2. 架构与数据流

1. 推荐生成时读取用户经纬度位置。
2. Weather Provider 调用 QWeather：
   - `now` 实时天气
   - `3d` 三日预报
   - `air/now` 实时空气质量
3. 实时天气写入 `weather_snapshots`（保持现有风险分级逻辑）。
4. 三日预报写入 `weather_forecasts`（每日一条）。
5. AI 输入增加 `forecasts[]`。

> 说明：QWeather 接口支持使用 `lon,lat` 直接查询，无需额外 geo lookup；如后续要缓存 LocationID，可再引入 geo lookup。

## 3. 数据模型

### 3.1 新表：`weather_forecasts`

**设计原则**
- 按 `user_id + forecast_date` 唯一。
- 字段尽可能覆盖 QWeather 3 天预报返回内容。
- 数值字段统一为国际标准单位，并标注单位语义。
- ~~所有字段均为 NOT NULL~~ 预报字段允许 NULL。

**表结构（建议）**

- `forecast_id` UUID PRIMARY KEY
- `user_id` TEXT NOT NULL
- `forecast_date` DATE NOT NULL
- `temp_max_c` DOUBLE PRECISION
- `temp_min_c` DOUBLE PRECISION
- `humidity` DOUBLE PRECISION  -- 百分比(0-100)
- `precip_mm` DOUBLE PRECISION
- `pressure_hpa` DOUBLE PRECISION
- `visibility_km` DOUBLE PRECISION
- `cloud_pct` DOUBLE PRECISION
- `uv_index` DOUBLE PRECISION
- `text_day` TEXT
- `text_night` TEXT
- `icon_day` TEXT
- `icon_night` TEXT
- `wind360_day` INT
- `wind_dir_day` TEXT
- `wind_scale_day` TEXT
- `wind_speed_day_ms` DOUBLE PRECISION  -- 由 km/h 转换
- `wind360_night` INT
- `wind_dir_night` TEXT
- `wind_scale_night` TEXT
- `wind_speed_night_ms` DOUBLE PRECISION  -- 由 km/h 转换
- `sunrise_time` TIME
- `sunset_time` TIME
- `moonrise_time` TIME
- `moonset_time` TIME
- `moon_phase` TEXT
- `moon_phase_icon` TEXT
- `created_at` TIMESTAMPTZ NOT NULL DEFAULT NOW()

**索引与约束**
- UNIQUE(`user_id`, `forecast_date`)

### 3.2 AI 输入结构扩展

`RecommendationWeather` 增加：

```json
{
  "forecasts": [
    {
      "date": "2026-03-11",
      "temp_max_c": 25.0,
      "temp_min_c": 12.0,
      "humidity": 55,
      "precip_mm": 0.0,
      "pressure_hpa": 1012,
      "visibility_km": 10,
      "cloud_pct": 20,
      "uv_index": 5,
      "text_day": "多云",
      "text_night": "晴",
      "icon_day": "101",
      "icon_night": "150",
      "wind360_day": 90,
      "wind_dir_day": "东风",
      "wind_scale_day": "3",
      "wind_speed_day_ms": 4.2,
      "wind360_night": 270,
      "wind_dir_night": "西风",
      "wind_scale_night": "2",
      "wind_speed_night_ms": 2.8,
      "sunrise_time": "06:30:00",
      "sunset_time": "18:20:00",
      "moonrise_time": "20:10:00",
      "moonset_time": "07:15:00",
      "moon_phase": "盈凸月",
      "moon_phase_icon": "804"
    }
  ]
}
```

> 说明：该数组允许为空（取数失败时）。

## 4. Weather Provider 设计

### 4.1 接口

- `GetSnapshot(ctx, location) (SnapshotInput, error)`
- `GetForecast(ctx, location) ([]ForecastInput, error)`

### 4.3 未配置行为

- 若 `api_key` 或 `api_host` 未配置，启动时打印日志并回退 mock provider（服务仍可启动）。

### 4.2 数据映射

- 温度、风速、紫外、湿度、气压、云量、能见度等字段按 QWeather 对应字段映射。
- 风速从 km/h 转为 m/s（除以 3.6）。
- 实时空气质量 `aqi` 仍注入快照，用于风险分级。

## 5. 错误兜底

- 实时天气请求失败：使用 `weather_snapshots` 最近一条。
- 预报失败：`forecasts` 置空。
- AI 调用失败：保持现有 fallback 输出策略。

## 6. 配置

新增配置段（不写死 Key，用户自行配置）：

```yaml
weather:
  qweather:
    api_key: "change-me"
    api_host: "ke487te4u4.re.qweatherapi.com"
    timeout_ms: 3000
```

## 7. 测试策略

- Provider 单测：
  - 模拟 QWeather 响应解析（now / 3d / air）。
  - 失败场景返回错误。
- 存储单测：
  - `weather_forecasts` 插入与查询。
- 推荐流程单测：
  - forecasts 成功写入 input_json。
  - 预报失败时 forecasts 为空。

## 8. 风险与注意事项

- QWeather 配额与超时：建议后续加入缓存与限流。
- 数据字段缺失：部分字段返回可能为空，解析时必须允许空值并设默认。
- 时间格式：`sunrise` / `sunset` 等返回为本地时间字符串，需严格解析。
