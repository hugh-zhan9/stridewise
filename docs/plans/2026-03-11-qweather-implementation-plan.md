# QWeather 天气接入与三日预报 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 接入 QWeather 实时天气与三日预报，落库 `weather_forecasts`，并将未来三天窗口写入推荐 AI 输入。

**Architecture:** 在 `internal/weather` 增加 QWeather Provider；`internal/storage` 增加 `weather_forecasts` 表与读写方法；推荐生成流程读取快照 + 预报并写入 input_json。

**Tech Stack:** Go, Kratos, Postgres, QWeather API

---

### Task 1: 扩展配置结构（weather.qweather）

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/config/config.yaml`

**Step 1: Write the failing test**

在 `config_test.go` 增加断言（新增 weather.qweather 字段）：

```go
if cfg.Weather.QWeather.APIKey != "key" {
	t.Fatalf("expected qweather api_key")
}
if cfg.Weather.QWeather.APIHost != "host" {
	t.Fatalf("expected qweather api_host")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/config -v`
Expected: FAIL（结构体缺字段或解析失败）

**Step 3: Write minimal implementation**

在 `Config` 中新增：

```go
Weather struct {
	QWeather struct {
		APIKey   string `yaml:"api_key"`
		APIHost  string `yaml:"api_host"`
		TimeoutMs int   `yaml:"timeout_ms"`
	} `yaml:"qweather"`
} `yaml:"weather"`
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/config -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/config/config.go backend/internal/config/config_test.go backend/config/config.yaml
git commit -m "配置新增QWeather"
```

---

### Task 2: 扩展 weather Provider 接口与类型

**Files:**
- Modify: `backend/internal/weather/provider.go`
- Modify: `backend/internal/weather/provider_test.go`

**Step 1: Write the failing test**

在 `provider_test.go` 增加测试：

```go
func TestMockProviderForecast(t *testing.T) {
	m := NewMockProvider(SnapshotInput{TemperatureC: 20}, []ForecastInput{{Date: time.Date(2026,3,11,0,0,0,0,time.UTC)}})
	out, err := m.GetForecast(context.Background(), Location{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 forecast")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/weather -v`
Expected: FAIL（接口缺失 / Mock 不支持）

**Step 3: Write minimal implementation**

- 新增 `ForecastInput` 结构体（包含日期、温度、风、降水、uv、描述等字段；~~aqi~~）
- `Provider` 增加 `GetForecast` 方法
- `MockProvider` 支持固定 forecasts

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/weather -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/weather/provider.go backend/internal/weather/provider_test.go
git commit -m "天气Provider支持三日预报"
```

---

### Task 3: 新增 weather_forecasts 表（迁移）

**Files:**
- Create: `backend/migrations/008_weather_forecasts.sql`
- Test: `backend/internal/storage/postgres_weather_forecast_test.go`

**Step 1: Write the failing test**

新增存储测试：

```go
func TestWeatherForecastStore(t *testing.T) {
	// 依赖 STRIDEWISE_TEST_DSN
	// 插入 3 天预报后读取验证
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`
Expected: FAIL（方法/表不存在）

**Step 3: Write minimal implementation**

新增迁移文件创建 `weather_forecasts` 表，字段尽可能详细（与设计一致，字段允许 NULL）。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`
Expected: PASS（需要测试库已应用迁移）

**Step 5: Commit**

```bash
git add backend/migrations/008_weather_forecasts.sql backend/internal/storage/postgres_weather_forecast_test.go
git commit -m "新增weather_forecasts表"
```

---

### Task 4: 实现 weather_forecasts 存储读写

**Files:**
- Modify: `backend/internal/storage/postgres.go`
- Modify: `backend/internal/storage/postgres_weather_forecast_test.go`

**Step 1: Write the failing test**

在测试中增加对方法的调用：

```go
forecasts := []WeatherForecast{...}
if err := store.UpsertWeatherForecasts(ctx, forecasts); err != nil { ... }
list, err := store.GetWeatherForecasts(ctx, "u1", from, to)
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`
Expected: FAIL（方法不存在）

**Step 3: Write minimal implementation**

- 新增 `WeatherForecast` 结构体
- 新增 `UpsertWeatherForecasts` 与 `GetWeatherForecasts`

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/storage/postgres.go backend/internal/storage/postgres_weather_forecast_test.go
git commit -m "新增天气预报存储读写"
```

---

### Task 5: 实现 QWeather Provider（解析 now/3d/air）

**Files:**
- Create: `backend/internal/weather/qweather.go`
- Create: `backend/internal/weather/qweather_test.go`

**Step 1: Write the failing test**

使用内置 JSON fixture 测试解析：

```go
func TestQWeatherParse(t *testing.T) {
	// now + 3d + air JSON
	// 断言 SnapshotInput 与 ForecastInput 映射
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/weather -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- 实现 HTTP client（超时来自 config）
- 解析 now / 3d / air JSON
- 返回 SnapshotInput + []ForecastInput
- 需要风速从 km/h 转 m/s

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/weather -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/weather/qweather.go backend/internal/weather/qweather_test.go
git commit -m "接入QWeather Provider"
```

---

### Task 6: 推荐流程写入预报与 AI 输入扩展

**Files:**
- Modify: `backend/internal/ai/recommender.go`
- Modify: `backend/internal/recommendation/processor.go`
- Modify: `backend/internal/recommendation/processor_test.go`
- Modify: `backend/internal/server/http_recommendation_test.go`（如有断言影响）

**Step 1: Write the failing test**

在推荐 processor 测试中断言 input_json 中含 forecasts：

```go
if !bytes.Contains(rec.InputJSON, []byte("forecasts")) {
	t.Fatalf("expected forecasts")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- `RecommendationWeather` 增加 `Forecasts` 字段
- `Processor.fetchWeather` 中调用 `provider.GetForecast`，写入 DB
- `Generate` 中将 forecasts 组装进 input
- 如果预报失败，forecasts 为空数组

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/ai/recommender.go backend/internal/recommendation/processor.go backend/internal/recommendation/processor_test.go

git commit -m "推荐输入包含三日预报"
```

---

### Task 7: 主程序接线（使用 QWeather Provider）

**Files:**
- Modify: `backend/cmd/api/main.go`

**Step 1: Write the failing test**

新增最小化测试（或通过集成测试验证）说明：

```go
// main 集成测试通常缺失，至少保证编译通过
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/cmd/api -v`
Expected: FAIL（若新增依赖未注入）

**Step 3: Write minimal implementation**

- 读取 config.weather.qweather
- 用 QWeatherProvider 替换 mock provider
- 若未配置 api_key，则打印日志并回退 mock provider

**Step 4: Run test to verify it passes**

Run: `go test ./backend/cmd/api -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/api/main.go

git commit -m "主程序接入QWeather"
```

---

### Task 8: 文档同步（如需）

**Files:**
- Modify: `docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`（可选）
- Modify: `docs/AI_CHANGELOG.md`

**Step 1: Write the failing test**

无（文档更新）

**Step 2: Run test to verify it fails**

无

**Step 3: Write minimal implementation**

- 如需要公开 JSON schema，补充 forecasts 字段说明

**Step 4: Run test to verify it passes**

无

**Step 5: Commit**

```bash
git add docs/...

git commit -m "文档补充三日预报字段"
```

---

## 执行前置条件

- STRIDEWISE_TEST_DSN 指向已应用最新迁移的测试数据库
- QWeather API Key 配置在 `config.yaml`（不提交）
