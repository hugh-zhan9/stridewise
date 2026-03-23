# 前后端 API 联调需求文档

本文档梳理了当前前端 MVP 界面中实际调用的后端接口及其入参和响应结构。
前端统一在请求中默认附加参数 `user_id=user_001` 和 Header `X-Internal-Token: stridewise-dev-token`。

所有响应结构被统一包装在一个 Envelope 中：
```json
{
  "data": { ... },
  "error": null,
  "meta": {
    "request_id": "string",
    "timestamp": "string",
    "confidence": 1.0
  }
}
```

---

## 1. 推荐与建议 (Recommendation)

### 1.1 获取今日推荐 (Get Latest Recommendation)
- **URL**: `GET /internal/v1/recommendations/latest`
- **Query Params**: `user_id` (string)
- **Request Body**: 无
- **Response `data` 结构** (Recommendation):
```typescript
{
  id: string;
  user_id: string;
  created_at: string;
  recommendation_date: string;
  rec_date_local: string;
  status: 'draft' | 'active' | 'consumed' | 'expired';
  should_run: boolean;
  workout_type: string;
  intensity_range: string;
  target_volume: string;
  suggested_time_window: string;
  risk_level: 'green' | 'yellow' | 'red';
  hydration_tip?: string;
  clothing_tip?: string;
  alternative_workouts?: {
    type: 'treadmill' | 'strength' | 'mobility' | 'rest';
    title: string;
    duration_min?: number;
    intensity?: 'low' | 'medium';
    tips?: string[];
  }[];
  explanation: string[];
  is_fallback: boolean;
  ai_provider?: string;
  ai_model?: string;
  engine_version?: string;
  prompt_version?: string;
  model_name?: string;
}
```

### 1.2 提交建议反馈 (Submit Recommendation Feedback)
- **URL**: `POST /internal/v1/recommendations/:id/feedback`
- **Path Params**: `id` (string, 建议ID)
- **Request Body**:
```typescript
{
  user_id: string;
  useful: 'useful' | 'neutral' | 'not_useful';
  reason?: string;
}
```
- **Response `data` 结构**: `{}` (或者返回创建的反馈对象)

---

## 2. 训练日志与基线 (Training & Baseline)

### 2.1 获取当前能力基线 (Get Current Baseline)
- **URL**: `GET /internal/v1/baseline/current`
- **Query Params**: `user_id` (string)
- **Request Body**: 无
- **Response `data` 结构** (BaselineResponse):
```typescript
{
  pace_zone: {
    easy: string;
    tempo: string;
    interval: string;
  };
  weekly_volume_range: {
    min_km: number;
    max_km: number;
  };
  recovery_level: 'high' | 'medium' | 'low';
  updated_at: string;
}
```

### 2.2 获取训练记录流水 (Get Training Logs)
- **URL**: `GET /internal/v1/training/logs`
- **Query Params**: `user_id` (string), `cursor` (string, 可选), `page_size` (number)
- **Request Body**: 无
- **Response `data` 结构** (TrainingLogListResponse 数组):
```typescript
[
  {
    log_id: string; // 后端返回 log_id，前端映射为 id
    start_time: string; // 后端返回 start_time，前端映射为 train_date_local
    training_type: string; // 后端返回 training_type，前端映射为 train_type
    duration_sec: number; // 后端返回秒，前端换算为 duration_min
    distance_km: number;
    pace_str: string; // 后端返回 pace_str，前端映射为 avg_pace
    rpe: number;
    discomfort: boolean; // 后端返回 discomfort，前端映射为 discomfort_flag
    source: 'manual' | 'third_party';
  }
]
```

### 2.3 新增手动训练记录 (Create Training Log)
- **URL**: `POST /internal/v1/training/logs`
- **Request Body**:
```typescript
{
  user_id: string;
  training_type: string;
  start_time: string; // e.g. "2026-03-12 00:00:00"
  duration: string; // e.g. "30m"
  distance_km: number;
  pace: string; // e.g. "06:00"
  rpe: number;
  discomfort: boolean;
}
```
- **Response `data` 结构**:
```typescript
{
  log_id: string;
  job_id: string;
}
```

---

## 3. 用户档案 (Profile)

### 3.1 获取当前用户档案 (Get Profile)
- **URL**: `GET /internal/v1/user/profile`
- **Query Params**: `user_id` (string)
- **Request Body**: 无
- **Response `data` 结构** (ProfileResponse):
```typescript
{
  user_id: string;
  goal_type: string;
  goal_target: string;
  ability_level: 'beginner' | 'intermediate' | 'advanced';
  ability_level_reason?: string;
  ability_level_updated_at?: string;
  resting_hr?: number;
  timezone: string;
  running_years: string;
  weekly_sessions: string;
  weekly_distance_km: string;
  longest_run_km: string;
  recent_discomfort: 'yes' | 'no';
}
```

### 3.2 初始化或全量更新用户档案 (Upsert Profile)
*说明：前端在设置/调整目标时复用了此接口*
- **URL**: `POST /internal/v1/user/profile`
- **Request Body**: 
（包含完整的 `userProfileRequest` 字段，前端目前主要为了覆盖 `goal_type` 和 `goal_target`）
```typescript
{
  user_id: string;
  goal_type: string;
  goal_target: string;
  // ... 其他必须的基础档案信息，基于后端的校验规则
}
```
- **Response `data` 结构** (同 Get Profile)