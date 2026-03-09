# StrideWise OpenAPI + AI JSON Schema

> 基于 `detailed-design-v2.md` 生成。目标：后端可据此实现接口，AI服务可据此完成结构化输入输出。

## 文档版本
- ~~当前版本：v1.1.0~~
- ~~当前版本：v1.2.0~~
- ~~当前版本：v1.3.0~~
- 当前版本：v1.4.0
- 发布日期：2026-03-09
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v1.4.0 | 2026-03-09 | 首发数据源调整为 Keep：provider 枚举新增 `keep` 并纳入统一来源范围。 |
| v1.3.0 | 2026-03-09 | 文档命名统一为 StrideWise；数据源枚举统一为 Strava/Garmin/Nike/GPX/TCX/FIT；明确当前阶段全部 API 为内部接口并统一使用 `X-Internal-Token`。 |
| v1.2.0 | 2026-03-09 | 对齐当前阶段安全策略：移除 JWT Bearer 默认鉴权，改为内部接口 `X-Internal-Token`；补充“完整用户鉴权后置”说明。 |
| v1.1.0 | 2026-03-09 | 新增版本管理区块；关联文档说明改为“主设计 + 附录关系”。 |
| v1.0.0 | 2026-03-09 | 首次发布 OpenAPI 3.1 与 AI 输入输出 JSON Schema。 |

## 0. 关联文档
- ~~主详细设计：`docs/plans/2026-03-09-running-ai-assistant-detailed-design-v2.md`~~
- ~~主详细设计（母文档）：`docs/plans/2026-03-09-running-ai-assistant-detailed-design-v2.md`~~
- 主详细设计（母文档）：`docs/plans/2026-03-09-stridewise-detailed-design-v2.md`
- ~~文档关系：本文件是详细设计的接口契约附录，专注 OpenAPI 与 AI Schema。~~
- 文档关系：本文件属于详细设计附录，专注 API 契约与 AI 输入输出 Schema，随主设计版本联动更新。

## 1. OpenAPI 3.1（MVP 草案）
- 当前阶段鉴权策略：
  - 业务 API 暂不启用完整用户鉴权（单用户/受控环境）
  - 当前阶段全部 API 均按内部接口治理，统一使用 `X-Internal-Token`
  - 进入多用户或 App 外放阶段后，补充 JWT/OAuth2 方案

```yaml
openapi: 3.1.0
info:
  title: StrideWise Internal API
  version: 1.0.0
  description: API for profile, training logs, recommendations, feedback, and import jobs.
servers:
  - url: https://api.example.com
security:
  - internalToken: []

tags:
  - name: profile
  - name: training
  - name: recommendation
  - name: import
  - name: internal

paths:
  /api/v1/profile/init:
    post:
      tags: [profile]
      summary: 初始化用户档案
      operationId: initProfile
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ProfileInitRequest'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ProfileResponse'
        '400':
          $ref: '#/components/responses/BadRequest'

  /api/v1/profile/goal:
    put:
      tags: [profile]
      summary: 更新用户目标
      operationId: updateGoal
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateGoalRequest'
      responses:
        '200':
          description: Updated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ProfileResponse'
        '400':
          $ref: '#/components/responses/BadRequest'

  /api/v1/profile/baseline:
    get:
      tags: [profile]
      summary: 获取当前能力基线
      operationId: getBaseline
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/BaselineResponse'

  /api/v1/training/logs:
    post:
      tags: [training]
      summary: 新增手动训练记录
      operationId: createTrainingLog
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateTrainingLogRequest'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TrainingLog'
        '400':
          $ref: '#/components/responses/BadRequest'

    get:
      tags: [training]
      summary: 查询训练记录（游标分页）
      operationId: listTrainingLogs
      parameters:
        - name: cursor
          in: query
          schema:
            type: string
        - name: page_size
          in: query
          schema:
            type: integer
            minimum: 1
            maximum: 100
            default: 20
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TrainingLogListResponse'

  /api/v1/training/import/{provider}/jobs:
    post:
      tags: [import]
      summary: 创建导入任务
      operationId: createImportJob
      parameters:
        - name: provider
          in: path
          required: true
          schema:
            type: string
            enum: [keep, strava, garmin, nike, gpx, tcx, fit]
      responses:
        '202':
          description: Accepted
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ImportJob'

  /api/v1/training/import/jobs/{jobId}:
    get:
      tags: [import]
      summary: 查询导入任务状态
      operationId: getImportJob
      parameters:
        - name: jobId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ImportJob'
        '404':
          $ref: '#/components/responses/NotFound'

  /api/v1/recommendations/today:
    get:
      tags: [recommendation]
      summary: 获取今日建议
      operationId: getTodayRecommendation
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Recommendation'
        '404':
          $ref: '#/components/responses/NotFound'

  /api/v1/recommendations/generate:
    post:
      tags: [recommendation]
      summary: 手动触发建议生成（内部）
      operationId: generateRecommendation
      security:
        - internalToken: []
      responses:
        '202':
          description: Accepted
          content:
            application/json:
              schema:
                type: object
                properties:
                  task_id:
                    type: string

  /api/v1/recommendations/{id}/consume:
    post:
      tags: [recommendation]
      summary: 标记建议已消费
      operationId: consumeRecommendation
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Recommendation'

  /api/v1/training/{id}/summary:
    get:
      tags: [training]
      summary: 获取训练总结
      operationId: getTrainingSummary
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TrainingSummary'

  /api/v1/recommendations/{id}/feedback:
    post:
      tags: [recommendation]
      summary: 提交建议反馈
      operationId: submitFeedback
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/FeedbackRequest'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Feedback'
        '409':
          description: Already submitted
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Problem'

  /internal/health:
    get:
      tags: [internal]
      summary: 健康检查（内网）
      security:
        - internalToken: []
      responses:
        '200':
          description: OK

  /internal/metrics:
    get:
      tags: [internal]
      summary: 指标（内网）
      security:
        - internalToken: []
      responses:
        '200':
          description: OK

components:
  securitySchemes:
    internalToken:
      type: apiKey
      in: header
      name: X-Internal-Token

  responses:
    BadRequest:
      description: Bad Request
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Problem'
    NotFound:
      description: Not Found
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Problem'

  schemas:
    Problem:
      type: object
      required: [type, title, status]
      properties:
        type: { type: string, format: uri }
        title: { type: string }
        status: { type: integer }
        detail: { type: string }
        instance: { type: string }

    ProfileInitRequest:
      type: object
      required: [gender, age, height_cm, weight_kg, timezone, goal_type]
      properties:
        gender: { type: string, enum: [male, female, other] }
        age: { type: integer, minimum: 10, maximum: 100 }
        height_cm: { type: number, minimum: 80, maximum: 250 }
        weight_kg: { type: number, minimum: 20, maximum: 300 }
        timezone: { type: string, example: Asia/Shanghai }
        default_location: { type: string, example: Shanghai }
        goal_type: { type: string, enum: [fat_loss, health_maintain, improve_5k] }
        goal_target: { type: string }

    UpdateGoalRequest:
      type: object
      required: [goal_type]
      properties:
        goal_type: { type: string, enum: [fat_loss, health_maintain, improve_5k] }
        goal_target: { type: string }

    ProfileResponse:
      type: object
      properties:
        user_id: { type: string }
        goal_type: { type: string }
        goal_target: { type: string }
        ability_level: { type: string, enum: [beginner, intermediate, advanced] }
        timezone: { type: string }

    BaselineResponse:
      type: object
      properties:
        pace_zone:
          type: object
          additionalProperties: true
        weekly_volume_range:
          type: object
          properties:
            min_km: { type: number }
            max_km: { type: number }
        recovery_level: { type: string, enum: [low, medium, high] }
        updated_at: { type: string, format: date-time }

    CreateTrainingLogRequest:
      type: object
      required: [train_date_local, train_type, duration_min, distance_km, rpe]
      properties:
        train_date_local: { type: string, format: date }
        train_type: { type: string, enum: [easy_run, long_run, interval, strength, stretch, rest] }
        duration_min: { type: integer, minimum: 1, maximum: 600 }
        distance_km: { type: number, minimum: 0 }
        avg_pace: { type: string, example: '05:30' }
        rpe: { type: integer, minimum: 1, maximum: 10 }
        discomfort_flag: { type: boolean, default: false }

    TrainingLog:
      type: object
      properties:
        id: { type: string }
        train_date_local: { type: string, format: date }
        train_type: { type: string }
        duration_min: { type: integer }
        distance_km: { type: number }
        avg_pace: { type: string }
        rpe: { type: integer }
        discomfort_flag: { type: boolean }
        source: { type: string, enum: [manual, third_party] }

    TrainingLogListResponse:
      type: object
      properties:
        items:
          type: array
          items:
            $ref: '#/components/schemas/TrainingLog'
        next_cursor: { type: string, nullable: true }

    ImportJob:
      type: object
      properties:
        id: { type: string }
        provider: { type: string }
        status: { type: string, enum: [queued, running, success, failed] }
        retry_count: { type: integer }
        error_code: { type: string, nullable: true }
        error_message: { type: string, nullable: true }

    Recommendation:
      type: object
      properties:
        id: { type: string }
        rec_date_local: { type: string, format: date }
        status: { type: string, enum: [draft, active, consumed, expired] }
        should_run: { type: boolean }
        workout_type: { type: string }
        intensity_range: { type: string }
        target_volume: { type: string }
        suggested_time_window: { type: string }
        risk_level: { type: string, enum: [green, yellow, red] }
        hydration_tip: { type: string }
        clothing_tip: { type: string }
        explanation:
          type: array
          items: { type: string }
        is_fallback: { type: boolean }
        engine_version: { type: string }
        prompt_version: { type: string }
        model_name: { type: string }

    TrainingSummary:
      type: object
      properties:
        training_log_id: { type: string }
        plan_match_score: { type: number, minimum: 0, maximum: 100 }
        completion_score: { type: number, minimum: 0, maximum: 100 }
        recovery_tip: { type: string }
        summary_json:
          type: object
          additionalProperties: true

    FeedbackRequest:
      type: object
      required: [usefulness]
      properties:
        usefulness: { type: string, enum: [useful, neutral, not_useful] }
        reason: { type: string, maxLength: 500 }

    Feedback:
      type: object
      properties:
        id: { type: string }
        recommendation_id: { type: string }
        usefulness: { type: string }
        reason: { type: string, nullable: true }
        created_at: { type: string, format: date-time }
```

## 2. AI 输入 JSON Schema（Recommendation Context）

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://example.com/schemas/ai-recommendation-input.schema.json",
  "title": "AIRecommendationInput",
  "type": "object",
  "required": [
    "request_id",
    "user_profile",
    "baseline",
    "weather",
    "training_load_7d",
    "constraints"
  ],
  "properties": {
    "request_id": { "type": "string" },
    "user_profile": {
      "type": "object",
      "required": ["ability_level", "goal_type", "timezone"],
      "properties": {
        "ability_level": { "type": "string", "enum": ["beginner", "intermediate", "advanced"] },
        "goal_type": { "type": "string", "enum": ["fat_loss", "health_maintain", "improve_5k"] },
        "age": { "type": "integer" },
        "weight_kg": { "type": "number" },
        "timezone": { "type": "string" },
        "default_location": { "type": "string" }
      },
      "additionalProperties": false
    },
    "baseline": {
      "type": "object",
      "required": ["recovery_level"],
      "properties": {
        "pace_zone": { "type": "object", "additionalProperties": true },
        "weekly_volume_range": {
          "type": "object",
          "properties": {
            "min_km": { "type": "number" },
            "max_km": { "type": "number" }
          },
          "required": ["min_km", "max_km"],
          "additionalProperties": false
        },
        "recovery_level": { "type": "string", "enum": ["low", "medium", "high"] }
      },
      "additionalProperties": false
    },
    "weather": {
      "type": "object",
      "required": ["today", "next_3_days", "risk_level"],
      "properties": {
        "today": {
          "type": "object",
          "required": ["temp", "feels_like", "humidity", "wind_speed", "precipitation_prob", "aqi", "uv_index"],
          "properties": {
            "temp": { "type": "number" },
            "feels_like": { "type": "number" },
            "humidity": { "type": "number" },
            "wind_speed": { "type": "number" },
            "precipitation_prob": { "type": "number" },
            "aqi": { "type": "number" },
            "uv_index": { "type": "number" }
          },
          "additionalProperties": false
        },
        "next_3_days": {
          "type": "array",
          "minItems": 3,
          "maxItems": 3,
          "items": {
            "type": "object",
            "required": ["date", "risk_level"],
            "properties": {
              "date": { "type": "string", "format": "date" },
              "risk_level": { "type": "string", "enum": ["green", "yellow", "red"] }
            },
            "additionalProperties": true
          }
        },
        "risk_level": { "type": "string", "enum": ["green", "yellow", "red"] }
      },
      "additionalProperties": false
    },
    "training_load_7d": {
      "type": "object",
      "required": ["sessions", "total_duration_min", "total_distance_km", "avg_rpe"],
      "properties": {
        "sessions": { "type": "integer", "minimum": 0 },
        "total_duration_min": { "type": "number", "minimum": 0 },
        "total_distance_km": { "type": "number", "minimum": 0 },
        "avg_rpe": { "type": "number", "minimum": 1, "maximum": 10 }
      },
      "additionalProperties": false
    },
    "recent_feedback": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "usefulness": { "type": "string", "enum": ["useful", "neutral", "not_useful"] },
          "reason": { "type": "string" }
        },
        "additionalProperties": false
      }
    },
    "constraints": {
      "type": "object",
      "required": ["hard_rules"],
      "properties": {
        "hard_rules": {
          "type": "array",
          "items": { "type": "string" }
        },
        "max_training_minutes": { "type": "integer", "minimum": 0 }
      },
      "additionalProperties": false
    }
  },
  "additionalProperties": false
}
```

## 3. AI 输出 JSON Schema（Recommendation Candidate）

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://example.com/schemas/ai-recommendation-output.schema.json",
  "title": "AIRecommendationOutput",
  "type": "object",
  "required": [
    "should_run",
    "workout_type",
    "intensity_range",
    "target_duration_or_distance",
    "suggested_time_window",
    "risk_level",
    "explanation",
    "confidence"
  ],
  "properties": {
    "should_run": { "type": "boolean" },
    "workout_type": {
      "type": "string",
      "enum": ["easy_run", "long_run", "interval", "strength", "stretch", "rest"]
    },
    "intensity_range": { "type": "string", "maxLength": 100 },
    "target_duration_or_distance": {
      "type": "object",
      "oneOf": [
        {
          "type": "object",
          "required": ["duration_min"],
          "properties": {
            "duration_min": { "type": "integer", "minimum": 0, "maximum": 240 }
          },
          "additionalProperties": false
        },
        {
          "type": "object",
          "required": ["distance_km"],
          "properties": {
            "distance_km": { "type": "number", "minimum": 0, "maximum": 60 }
          },
          "additionalProperties": false
        }
      ]
    },
    "suggested_time_window": { "type": "string", "maxLength": 100 },
    "hydration_tip": { "type": "string", "maxLength": 300 },
    "clothing_tip": { "type": "string", "maxLength": 300 },
    "risk_level": { "type": "string", "enum": ["green", "yellow", "red"] },
    "explanation": {
      "type": "array",
      "minItems": 2,
      "maxItems": 6,
      "items": { "type": "string", "maxLength": 300 }
    },
    "confidence": { "type": "number", "minimum": 0, "maximum": 1 },
    "warnings": {
      "type": "array",
      "items": { "type": "string", "maxLength": 200 }
    }
  },
  "additionalProperties": false
}
```

## 4. 校验与落地建议
- 先做 Schema 校验，再入库。
- AI 输出若不满足 Schema：直接走 Rule-only fallback。
- AI 输出满足 Schema 但违反硬规则：规则引擎覆盖字段并记录 `override_reason`。
- 建议在 `recommendation` 表记录 `prompt_version`、`model_name`、`engine_version` 用于回溯。
