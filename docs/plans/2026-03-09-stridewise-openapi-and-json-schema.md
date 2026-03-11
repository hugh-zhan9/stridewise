# StrideWise OpenAPI + AI JSON Schema

> 基于 `detailed-design-v2.md` 生成。目标：后端可据此实现接口，AI服务可据此完成结构化输入输出。

## 文档版本
- ~~当前版本：v1.1.0~~
- ~~当前版本：v1.2.0~~
- ~~当前版本：v1.3.0~~
- ~~当前版本：v1.4.0~~
- ~~当前版本：v1.5.0~~
- ~~当前版本：v1.6.0~~
- ~~当前版本：v1.7.0~~
- ~~当前版本：v1.8.0~~
- ~~当前版本：v1.9.0~~
- ~~当前版本：v1.10.0~~
- ~~当前版本：v1.11.0~~
- ~~当前版本：v1.12.0~~
- 当前版本：v1.13.0
- 发布日期：2026-03-11
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v1.13.0 | 2026-03-11 | 新增滚动 7/30 天训练趋势分析内部接口与 schema。 |
| v1.12.0 | 2026-03-11 | AI 输出解释字段移除 minItems 约束，允许少于 2 条；说明 explanation 条数不作为失败条件。 |
| v1.11.0 | 2026-03-11 | Profile 增加静息心率；Forecast 增加预测 AQI 字段与来源。 |
| v1.10.0 | 2026-03-11 | AI 推荐输入新增 latest_training_feedback 字段。 |
| v1.9.0 | 2026-03-11 | 内部接口响应统一包裹 Envelope，标准化错误与兜底元信息字段。 |
| v1.8.0 | 2026-03-11 | 能力层级改为 AI 自动判定，Profile 响应新增能力层级元信息。 |
| v1.7.0 | 2026-03-11 | 用户问卷字段落库，Profile schema 增加问卷字段。 |
| v1.6.0 | 2026-03-10 | 训练总结/反馈支持 source_type/source_id，新增训练反馈内部接口说明。 |
| v1.5.0 | 2026-03-10 | 新增内部建议生成/反馈接口与 Recommendation 扩展字段。 |
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

## 字段变更提示
- ~~TrainingSummary.training_log_id / plan_match_score / completion_score / recovery_tip / summary_json~~ → 使用结构化字段并新增 `source_type/source_id`
- ~~Profile 未包含 resting_hr~~ → ProfileInitRequest/ProfileResponse 增加 `resting_hr`（可选）
- ~~Forecast 未包含 AQI~~ → RecommendationForecast 增加 `aqi`/`aqi_source`
- ~~AI 输出 explanation 至少 2 条~~ → 解释条数改为期望值，移除 minItems 限制

## 1. OpenAPI 3.1（MVP 草案）
- 当前阶段鉴权策略：
  - 业务 API 暂不启用完整用户鉴权（单用户/受控环境）
  - 当前阶段全部 API 均按内部接口治理，统一使用 `X-Internal-Token`
  - 进入多用户或 App 外放阶段后，补充 JWT/OAuth2 方案
- 响应标准化：
  - 内部接口响应统一 Envelope 包裹（`data/error/meta`）
  - 本文档中 response `schema` 表达 `data` 字段的业务结构

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
      deprecated: true
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
      deprecated: true
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
      deprecated: true
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
      deprecated: true
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

  /internal/v1/recommendations/generate:
    post:
      tags: [recommendation]
      summary: 生成建议（内部）
      operationId: generateRecommendationInternal
      security:
        - internalToken: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/RecommendationGenerateRequest'
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Recommendation'

  /internal/v1/recommendations/latest:
    get:
      tags: [recommendation]
      summary: 获取最新建议（内部）
      operationId: getLatestRecommendationInternal
      security:
        - internalToken: []
      parameters:
        - name: user_id
          in: query
          required: true
          schema: { type: string }
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Recommendation'

  /internal/v1/recommendations/{id}/feedback:
    post:
      tags: [recommendation]
      summary: 提交建议反馈（内部）
      operationId: submitRecommendationFeedbackInternal
      security:
        - internalToken: []
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: string }
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/RecommendationFeedbackRequest'
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/RecommendationFeedback'

  /internal/v1/training/summaries:
    get:
      tags: [training]
      summary: 获取训练总结列表（内部）
      operationId: listTrainingSummariesInternal
      security:
        - internalToken: []
      parameters:
        - name: user_id
          in: query
          required: true
          schema: { type: string }
        - name: from
          in: query
          required: false
          schema: { type: string, format: date-time }
        - name: to
          in: query
          required: false
          schema: { type: string, format: date-time }
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/TrainingSummary'

  /internal/v1/trends:
    get:
      tags: [training]
      summary: 获取滚动训练趋势（内部）
      operationId: getRollingTrendsInternal
      security:
        - internalToken: []
      parameters:
        - name: user_id
          in: query
          required: true
          schema: { type: string }
        - name: window
          in: query
          required: true
          schema:
            type: string
            enum: [7d, 30d]
        - name: as_of
          in: query
          required: false
          schema: { type: string, example: "2026-03-10 12:00:00" }
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TrendResult'

  /internal/v1/training/feedback:
    post:
      tags: [training]
      summary: 提交训练反馈（内部）
      operationId: submitTrainingFeedbackInternal
      security:
        - internalToken: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/TrainingFeedbackRequest'
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TrainingFeedbackResponse'

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
            $ref: '#/components/schemas/Envelope'
    NotFound:
      description: Not Found
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Envelope'

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

    EnvelopeError:
      type: object
      required: [code, message]
      properties:
        code:
          type: string
          enum: [bad_request, unauthorized, forbidden, not_found, conflict, dependency_unavailable, internal_error]
        message: { type: string }

    EnvelopeMeta:
      type: object
      required: [request_id, timestamp, confidence]
      properties:
        request_id: { type: string }
        timestamp: { type: string, format: date-time }
        fallback_reason:
          type: string
          enum: [insufficient_data, weather_api_failed, ai_unavailable, ability_level_not_ready, safety_override, third_party_import_failed, field_missing]
        confidence: { type: number, minimum: 0, maximum: 1 }

    Envelope:
      type: object
      required: [data, error, meta]
      properties:
        data: {}
        error:
          $ref: '#/components/schemas/EnvelopeError'
          nullable: true
        meta:
          $ref: '#/components/schemas/EnvelopeMeta'

    ProfileInitRequest:
      type: object
      required: [gender, age, height_cm, weight_kg, timezone, goal_type, running_years, weekly_sessions, weekly_distance_km, longest_run_km, recent_discomfort]
      properties:
        gender: { type: string, enum: [male, female, other] }
        age: { type: integer, minimum: 10, maximum: 100 }
        height_cm: { type: number, minimum: 80, maximum: 250 }
        weight_kg: { type: number, minimum: 20, maximum: 300 }
        resting_hr: { type: integer, minimum: 30, maximum: 120 }
        timezone: { type: string, example: Asia/Shanghai }
        default_location: { type: string, example: Shanghai }
        goal_type: { type: string, enum: [fat_loss, health_maintain, improve_5k] }
        goal_target: { type: string }
        running_years: { type: string, enum: ["0", "<1", "1-3", "3+"] }
        weekly_sessions: { type: string, enum: ["0-1", "2-3", "4+"] }
        weekly_distance_km: { type: string, enum: ["0-5", "5-15", "15-30", "30+"] }
        longest_run_km: { type: string, enum: ["0", "3", "5", "10", "21"] }
        recent_discomfort: { type: string, enum: [yes, no] }

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
        ability_level_reason: { type: string, nullable: true }
        ability_level_updated_at: { type: string, format: date-time, nullable: true }
        resting_hr: { type: integer, nullable: true }
        timezone: { type: string }
        running_years: { type: string, enum: ["0", "<1", "1-3", "3+"] }
        weekly_sessions: { type: string, enum: ["0-1", "2-3", "4+"] }
        weekly_distance_km: { type: string, enum: ["0-5", "5-15", "15-30", "30+"] }
        longest_run_km: { type: string, enum: ["0", "3", "5", "10", "21"] }
        recent_discomfort: { type: string, enum: [yes, no] }

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
        user_id: { type: string }
        created_at: { type: string, format: date-time }
        recommendation_date: { type: string, format: date }
        input_json:
          type: object
          additionalProperties: true
        output_json:
          type: object
          additionalProperties: true
        override_json:
          type: object
          additionalProperties: true
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
        alternative_workouts:
          type: array
          items:
            type: object
            required: [type, title]
            properties:
              type: { type: string, enum: [treadmill, strength, mobility, rest] }
              title: { type: string }
              duration_min: { type: integer }
              intensity: { type: string, enum: [low, medium] }
              tips:
                type: array
                items: { type: string }
            additionalProperties: false
        explanation:
          type: array
          items: { type: string }
        is_fallback: { type: boolean }
        ai_provider: { type: string }
        ai_model: { type: string }
        engine_version: { type: string }
        prompt_version: { type: string }
        model_name: { type: string }

    TrainingSummary:
      type: object
      properties:
        summary_id: { type: string }
        user_id: { type: string }
        source_type: { type: string, enum: [log, activity] }
        source_id: { type: string }
        log_id: { type: string, nullable: true }
        completion_rate: { type: string }
        intensity_match: { type: string }
        recovery_advice: { type: string }
        anomaly_notes: { type: string }
        performance_notes: { type: string }
        next_suggestion: { type: string }
        deleted_at: { type: string, format: date-time, nullable: true }
        created_at: { type: string, format: date-time }
        updated_at: { type: string, format: date-time }

    TrendResult:
      type: object
      properties:
        window_start: { type: string, format: date }
        window_end: { type: string, format: date }
        summary:
          $ref: '#/components/schemas/TrendSummary'
        series:
          type: array
          items:
            $ref: '#/components/schemas/TrendPoint'

    TrendSummary:
      type: object
      properties:
        sessions: { type: integer }
        distance_km: { type: number }
        duration_sec: { type: integer }
        avg_pace_sec_per_km: { type: integer }
        avg_rpe: { type: number }
        summary_count: { type: integer }
        completion_rate_dist:
          type: object
          additionalProperties: { type: integer }
        intensity_match_dist:
          type: object
          additionalProperties: { type: integer }
        recovery_advice_tags:
          type: object
          additionalProperties: { type: integer }
        acwr_srpe: { type: number, nullable: true }
        acwr_distance: { type: number, nullable: true }
        monotony: { type: number, nullable: true }
        strain: { type: number, nullable: true }

    TrendPoint:
      type: object
      properties:
        date: { type: string, format: date }
        sessions: { type: integer }
        distance_km: { type: number }
        duration_sec: { type: integer }
        avg_pace_sec_per_km: { type: integer }
        avg_rpe: { type: number }

    FeedbackRequest:
      type: object
      required: [usefulness]
      properties:
        usefulness: { type: string, enum: [useful, neutral, not_useful] }
        reason: { type: string, maxLength: 500 }

    TrainingFeedbackRequest:
      type: object
      required: [user_id, source_type, source_id, content]
      properties:
        user_id: { type: string }
        source_type: { type: string, enum: [log, activity] }
        source_id: { type: string }
        content: { type: string, maxLength: 1000 }

    TrainingFeedbackResponse:
      type: object
      properties:
        feedback_id: { type: string }

    Feedback:
      type: object
      properties:
        id: { type: string }
        recommendation_id: { type: string }
        usefulness: { type: string }
        reason: { type: string, nullable: true }
        created_at: { type: string, format: date-time }

    RecommendationGenerateRequest:
      type: object
      required: [user_id]
      properties:
        user_id: { type: string }

    RecommendationFeedbackRequest:
      type: object
      required: [useful]
      properties:
        useful: { type: string, enum: [yes, neutral, no] }
        reason: { type: string, maxLength: 500 }

    RecommendationFeedback:
      type: object
      properties:
        feedback_id: { type: string }
        rec_id: { type: string }
        user_id: { type: string }
        useful: { type: string }
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
    "constraints",
    "recovery_status"
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
      "required": ["temperature_c", "feels_like_c", "humidity", "wind_speed_ms", "precipitation_prob", "aqi", "uv_index", "risk_level", "forecasts"],
      "properties": {
        "temperature_c": { "type": "number" },
        "feels_like_c": { "type": "number" },
        "humidity": { "type": "number" },
        "wind_speed_ms": { "type": "number" },
        "precipitation_prob": { "type": "number" },
        "aqi": { "type": "number" },
        "uv_index": { "type": "number" },
        "risk_level": { "type": "string" },
        "forecasts": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["date", "aqi", "aqi_source"],
            "properties": {
              "date": { "type": "string", "format": "date" },
              "temp_max_c": { "type": ["number", "null"] },
              "temp_min_c": { "type": ["number", "null"] },
              "humidity": { "type": ["number", "null"] },
              "precip_mm": { "type": ["number", "null"] },
              "pressure_hpa": { "type": ["number", "null"] },
              "visibility_km": { "type": ["number", "null"] },
              "cloud_pct": { "type": ["number", "null"] },
              "uv_index": { "type": ["number", "null"] },
              "aqi": { "type": "integer" },
              "aqi_source": { "type": "string", "enum": ["local", "qaqi"] },
              "text_day": { "type": ["string", "null"] },
              "text_night": { "type": ["string", "null"] },
              "icon_day": { "type": ["string", "null"] },
              "icon_night": { "type": ["string", "null"] },
              "wind360_day": { "type": ["integer", "null"] },
              "wind_dir_day": { "type": ["string", "null"] },
              "wind_scale_day": { "type": ["string", "null"] },
              "wind_speed_day_ms": { "type": ["number", "null"] },
              "wind360_night": { "type": ["integer", "null"] },
              "wind_dir_night": { "type": ["string", "null"] },
              "wind_scale_night": { "type": ["string", "null"] },
              "wind_speed_night_ms": { "type": ["number", "null"] },
              "sunrise_time": { "type": ["string", "null"], "format": "time" },
              "sunset_time": { "type": ["string", "null"], "format": "time" },
              "moonrise_time": { "type": ["string", "null"], "format": "time" },
              "moonset_time": { "type": ["string", "null"], "format": "time" },
              "moon_phase": { "type": ["string", "null"] },
              "moon_phase_icon": { "type": ["string", "null"] }
            },
            "additionalProperties": false
          }
        }
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
    },
    "latest_training_feedback": {
      "type": "object",
      "required": ["source_type", "source_id", "content", "summary"],
      "properties": {
        "source_type": { "type": "string", "enum": ["log", "activity"] },
        "source_id": { "type": "string" },
        "created_at": { "type": "string", "format": "date-time" },
        "content": { "type": "string" },
        "summary": {
          "type": "object",
          "required": ["completion_rate", "intensity_match", "recovery_advice", "anomaly_notes", "performance_notes", "next_suggestion"],
          "properties": {
            "completion_rate": { "type": "string" },
            "intensity_match": { "type": "string" },
            "recovery_advice": { "type": "string" },
            "anomaly_notes": { "type": "string" },
            "performance_notes": { "type": "string" },
            "next_suggestion": { "type": "string" }
          },
          "additionalProperties": false
        }
      },
      "additionalProperties": false
    },
    "recovery_status": { "type": "string", "enum": ["green", "yellow", "red"] }
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
    "alternative_workouts": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["type", "title"],
        "properties": {
          "type": { "type": "string", "enum": ["treadmill", "strength", "mobility", "rest"] },
          "title": { "type": "string", "maxLength": 100 },
          "duration_min": { "type": "integer", "minimum": 0, "maximum": 180 },
          "intensity": { "type": "string", "enum": ["low", "medium"] },
          "tips": {
            "type": "array",
            "items": { "type": "string", "maxLength": 200 }
          }
        },
        "additionalProperties": false
      }
    },
    "risk_level": { "type": "string", "enum": ["green", "yellow", "red"] },
    "explanation": {
      "type": "array",
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
- ~~AI 输出若不满足 Schema：直接走 Rule-only fallback。~~
- AI 输出若不满足 Schema：直接走 Rule-only fallback（explanation 条数不作为失败条件）。
- AI 输出满足 Schema 但违反硬规则：规则引擎覆盖字段并记录 `override_reason`。
- 建议在 `recommendation` 表记录 `prompt_version`、`model_name`、`engine_version` 用于回溯。
