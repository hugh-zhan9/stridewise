# 静息心率 + 预测 AQI + 解释下限 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 P0 增补静息心率字段、预测 AQI 入参与落库，并强制建议解释不少于 2 条。

**Architecture:** 扩展用户档案与天气预测数据模型；QWeather 增加空气质量日预报调用并按日期合并到天气预测；推荐生成流程要求预测 AQI 与解释下限，缺失则触发保守建议。

**Tech Stack:** Go、PostgreSQL、Asynq、QWeather API

---

### Task 1: 文档与需求对齐（非代码）

**Files:**
- Modify: `request.md`
- Modify: `docs/plans/2026-03-10-user-weather-design.md`
- Modify: `docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`

**Step 1: 更新 request.md**
- 增加“静息心率（可选）”字段说明。
- 明确“预测 AQI 必需（本地优先，缺失用 QAQI）”。
- 明确“AI 建议解释至少 2 条”。

**Step 2: 更新设计文档版本与删除线**
- `2026-03-10-user-weather-design.md`：新增版本记录；对旧表述用删除线标注，补充预测 AQI 逻辑。
- `2026-03-09-stridewise-openapi-and-json-schema.md`：新增版本记录；对变更字段（Profile + Forecast + AI 输入）用删除线标注旧描述并补充新字段。

---

### Task 2: 用户档案新增静息心率（TDD）

**Files:**
- Create: `backend/migrations/010_add_user_profile_resting_hr.sql`
- Modify: `backend/internal/storage/postgres.go`
- Modify: `backend/internal/server/http.go`
- Test: `backend/internal/storage/postgres_user_weather_test.go`
- Test: `backend/internal/server/http_user_weather_test.go`

**Step 1: Write the failing test**

在 `postgres_user_weather_test.go` 补充：

```go
func TestUserProfileRestingHR(t *testing.T) {
    dsn := os.Getenv("STRIDEWISE_TEST_DSN")
    if dsn == "" {
        t.Skip("STRIDEWISE_TEST_DSN not set")
    }
    pool, _ := pgxpool.New(context.Background(), dsn)
    defer pool.Close()

    store := NewPostgresStore(pool)
    profile := UserProfile{
        UserID: "u-rest-1",
        Gender: "male",
        Age: 30,
        HeightCM: 175,
        WeightKG: 70,
        GoalType: "health",
        GoalCycle: "12w",
        GoalFrequency: 3,
        GoalPace: "05'30''",
        RunningYears: "1-3",
        WeeklySessions: "2-3",
        WeeklyDistanceKM: "5-15",
        LongestRunKM: "5-10",
        RecentDiscomfort: "no",
        LocationLat: 39.9,
        LocationLng: 116.4,
        Country: "CN",
        Province: "BJ",
        City: "Beijing",
        LocationSource: "manual",
        RestingHR: 55,
    }
    if err := store.UpsertUserProfile(context.Background(), profile); err != nil {
        t.Fatalf("upsert failed: %v", err)
    }
    got, err := store.GetUserProfile(context.Background(), profile.UserID)
    if err != nil {
        t.Fatalf("get failed: %v", err)
    }
    if got.RestingHR != 55 {
        t.Fatalf("expected resting_hr 55, got %d", got.RestingHR)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`  
Expected: FAIL（字段/迁移缺失）

**Step 3: Write minimal implementation**

- 新增迁移 `010_add_user_profile_resting_hr.sql`：
  - `ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS resting_hr INT;`
- `storage.UserProfile` 增加 `RestingHR int`。
- `UpsertUserProfile/GetUserProfile` 增加字段读写。
- `userProfileRequest` 增加 `resting_hr`，`validateUserProfileRequest` 接受可选数值。
- `http_user_weather_test.go` 增加请求包含 `resting_hr` 的用例并断言写入。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`  
Run: `go test ./backend/internal/server -v`

**Step 5: Commit**

```bash
git add backend/migrations/010_add_user_profile_resting_hr.sql \
        backend/internal/storage/postgres.go \
        backend/internal/server/http.go \
        backend/internal/storage/postgres_user_weather_test.go \
        backend/internal/server/http_user_weather_test.go

git commit -m "新增用户静息心率字段"
```

---

### Task 3: 预测 AQI 落库字段（TDD）

**Files:**
- Create: `backend/migrations/011_add_weather_forecast_aqi.sql`
- Modify: `backend/internal/storage/postgres.go`
- Test: `backend/internal/storage/postgres_weather_forecast_test.go`

**Step 1: Write the failing test**

在 `postgres_weather_forecast_test.go` 增加列存在性检查：

```go
func TestWeatherForecastAQIColumns(t *testing.T) {
    dsn := os.Getenv("STRIDEWISE_TEST_DSN")
    if dsn == "" {
        t.Skip("STRIDEWISE_TEST_DSN not set")
    }
    pool, _ := pgxpool.New(context.Background(), dsn)
    defer pool.Close()

    var col string
    err := pool.QueryRow(context.Background(),
        "SELECT column_name FROM information_schema.columns WHERE table_name='weather_forecasts' AND column_name='aqi_local'").Scan(&col)
    if err != nil || col != "aqi_local" {
        t.Fatalf("aqi_local column missing")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`  
Expected: FAIL（列缺失）

**Step 3: Write minimal implementation**

- 新增迁移 `011_add_weather_forecast_aqi.sql`：
  - `aqi_local INT NULL`
  - `aqi_qaqi INT NULL`
  - `aqi_source TEXT NOT NULL DEFAULT ''`
- `storage.WeatherForecast` 增加 `AQILocal *int`, `AQIQAQI *int`, `AQISource *string`
- `UpsertWeatherForecasts` / `GetWeatherForecasts` 增加字段写读

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`

**Step 5: Commit**

```bash
git add backend/migrations/011_add_weather_forecast_aqi.sql \
        backend/internal/storage/postgres.go \
        backend/internal/storage/postgres_weather_forecast_test.go

git commit -m "新增天气预测 AQI 字段"
```

---

### Task 4: QWeather 空气质量日预报接入（TDD）

**Files:**
- Modify: `backend/internal/weather/forecast.go`
- Modify: `backend/internal/weather/provider.go`
- Modify: `backend/internal/weather/qweather.go`
- Test: `backend/internal/weather/qweather_test.go`

**Step 1: Write the failing test**

在 `qweather_test.go` 增加：

```go
func TestParseAirDailyForecasts(t *testing.T) {
    payload := []byte(`{"days":[{"forecastStartTime":"2026-03-11T00:00Z","indexes":[{"code":"local","aqi":80},{"code":"qaqi","aqi":60}]}]}`)
    days, err := parseQWeatherAirDaily(payload)
    if err != nil { t.Fatalf("parse failed: %v", err) }
    if len(days) != 1 || days[0].AQILocal == nil || *days[0].AQILocal != 80 {
        t.Fatalf("expected local aqi")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/weather -v`  
Expected: FAIL（解析函数不存在）

**Step 3: Write minimal implementation**

- 扩展 `ForecastInput` 增加：`AQILocal *int`, `AQIQAQI *int`, `AQISource *string`
- 在 `qweather.go` 增加 `fetchAirDailyForecasts`：
  - 调用 `/airquality/v1/daily/{lat}/{lon}`，解析 `days[].indexes[]` 中 `code` 与 `aqi`
  - 按日期合并到天气预报结果（本地 AQI 优先，缺失用 QAQI）
  - 若预测 AQI 缺失，返回错误（触发保守建议）
- `GetForecast` 调用天气预报 + 空气质量预报并合并

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/weather -v`

**Step 5: Commit**

```bash
git add backend/internal/weather/forecast.go \
        backend/internal/weather/provider.go \
        backend/internal/weather/qweather.go \
        backend/internal/weather/qweather_test.go

git commit -m "接入空气质量日预报并合并预测 AQI"
```

---

### Task 5: 推荐输入加入预测 AQI（TDD）

**Files:**
- Modify: `backend/internal/ai/recommender.go`
- Modify: `backend/internal/recommendation/processor.go`
- Modify: `backend/internal/recommendation/processor_test.go`

**Step 1: Write the failing test**

```go
func TestRecommendationForecastAQIRequired(t *testing.T) {
    store := &fakeStore{profile: sampleProfile(), baseline: sampleBaseline()}
    provider := fakeWeather{forecasts: []weather.ForecastInput{{Date: time.Now()}}}
    p := NewProcessor(store, provider, fakeAI{})
    _, err := p.Generate(context.Background(), "u1")
    if err == nil {
        t.Fatalf("expected error when forecast AQI missing")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`  
Expected: FAIL（未校验 AQI）

**Step 3: Write minimal implementation**

- `ai.RecommendationForecast` 增加 `aqi` 与 `aqi_source`
- `mapForecasts` 增加 AQI 映射
- `fetchForecasts` 返回错误时视为天气错误（触发保守建议）

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`

**Step 5: Commit**

```bash
git add backend/internal/ai/recommender.go \
        backend/internal/recommendation/processor.go \
        backend/internal/recommendation/processor_test.go

git commit -m "推荐输入加入预测 AQI"
```

---

### Task 6: 建议解释至少 2 条（TDD）

**Files:**
- Modify: `backend/internal/recommendation/processor.go`
- Modify: `backend/internal/recommendation/processor_test.go`

**Step 1: Write the failing test**

```go
func TestRecommendationExplanationMinTwo(t *testing.T) {
    out := RecommendationOutput{Explanation: []string{"one"}}
    got := ensureExplanation(out)
    if len(got.Explanation) < 2 {
        t.Fatalf("expected at least 2 explanations")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`  
Expected: FAIL（函数不存在）

**Step 3: Write minimal implementation**

- 新增 `ensureExplanation`，不足 2 条时补充默认原因
- 在 `Generate` 中对输出统一调用该函数（含保守建议）

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`

**Step 5: Commit**

```bash
git add backend/internal/recommendation/processor.go \
        backend/internal/recommendation/processor_test.go

git commit -m "强制建议解释至少两条"
```

---

### Task 7: OpenAPI + JSON Schema 更新（非代码）

**Files:**
- Modify: `docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`

**Step 1: 更新 Profile 与 Forecast schema**
- `ProfileInitRequest`/`ProfileResponse` 增加 `resting_hr`
- `RecommendationForecast` 增加 `aqi`/`aqi_source`
- AI Input/Output JSON Schema 同步字段

---

### Task 8: 全量回归与飞行记录

**Step 1: Run full tests**

Run: `go test ./backend/...`  
Expected: PASS

**Step 2: Flight Recorder**

```bash
python3 /Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py "Feature" "补齐静息心率与预测AQI并强化解释约束" "新增外部依赖调用与字段变更，可能影响建议生成与天气依赖稳定性" "S2"
```

**Step 3: Commit**

```bash
git add docs/AI_CHANGELOG.md
git commit -m "记录静息心率与预测AQI变更"
```
