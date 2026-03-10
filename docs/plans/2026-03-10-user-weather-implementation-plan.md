# User Profile + Weather Module Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现用户档案（含位置必填）、天气快照与风险分级的内部接口与存储，为“AI 优先 + 规则兜底”提供可靠输入。

**Architecture:** 以 `internal/weather` 提供风险分级与 Provider 接口，`internal/storage` 负责档案/快照持久化，`internal/server` 提供内部 HTTP 接口并做校验与兜底。

**Tech Stack:** Go、Kratos HTTP、PostgreSQL（pgx）、Asynq、Docker Compose

---

### Task 1: 天气风险分级核心函数

**Files:**
- Create: `backend/internal/weather/risk.go`
- Create: `backend/internal/weather/risk_test.go`

**Step 1: Write the failing test**

```go
package weather

import "testing"

func TestClassifyRisk_RedByAQI(t *testing.T) {
	input := SnapshotInput{
		TemperatureC:      22,
		FeelsLikeC:        22,
		Humidity:          0.5,
		WindSpeedMS:       3,
		PrecipitationProb: 0.1,
		AQI:               180,
		UVIndex:           3,
	}
	if got := ClassifyRisk(input); got != RiskRed {
		t.Fatalf("expected %s, got %s", RiskRed, got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/weather -v`  
Expected: FAIL (undefined: SnapshotInput / ClassifyRisk)

**Step 3: Write minimal implementation**

```go
package weather

type RiskLevel string

const (
	RiskGreen  RiskLevel = "green"
	RiskYellow RiskLevel = "yellow"
	RiskRed    RiskLevel = "red"
)

type SnapshotInput struct {
	TemperatureC      float64
	FeelsLikeC        float64
	Humidity          float64
	WindSpeedMS       float64
	PrecipitationProb float64
	AQI               int
	UVIndex           float64
}

func ClassifyRisk(input SnapshotInput) RiskLevel {
	if input.FeelsLikeC >= 40.6 {
		return RiskRed
	}
	if input.WindSpeedMS >= 17.9 {
		return RiskRed
	}
	if input.AQI >= 151 {
		return RiskRed
	}
	if input.UVIndex >= 8 {
		return RiskRed
	}
	if input.FeelsLikeC >= 32.2 {
		return RiskYellow
	}
	if input.WindSpeedMS >= 13.9 {
		return RiskYellow
	}
	if input.PrecipitationProb >= 0.4 {
		return RiskYellow
	}
	if input.AQI >= 101 {
		return RiskYellow
	}
	if input.UVIndex >= 3 {
		return RiskYellow
	}
	return RiskGreen
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/weather -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/weather/risk.go backend/internal/weather/risk_test.go
git commit -m "新增天气风险分级核心逻辑"
```

---

### Task 2: Weather Provider 接口与 Mock 实现

**Files:**
- Create: `backend/internal/weather/provider.go`
- Create: `backend/internal/weather/provider_test.go`

**Step 1: Write the failing test**

```go
package weather

import (
	"context"
	"testing"
)

func TestMockProvider_ReturnsFixedSnapshot(t *testing.T) {
	mock := NewMockProvider(SnapshotInput{TemperatureC: 18})
	got, err := mock.GetSnapshot(context.Background(), Location{Lat: 1, Lng: 2})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.TemperatureC != 18 {
		t.Fatalf("expected 18, got %v", got.TemperatureC)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/weather -v`  
Expected: FAIL (undefined: NewMockProvider / Location)

**Step 3: Write minimal implementation**

```go
package weather

import "context"

type Location struct {
	Lat      float64
	Lng      float64
	Country  string
	Province string
	City     string
}

type Provider interface {
	GetSnapshot(ctx context.Context, location Location) (SnapshotInput, error)
}

type MockProvider struct {
	fixed SnapshotInput
}

func NewMockProvider(fixed SnapshotInput) MockProvider {
	return MockProvider{fixed: fixed}
}

func (m MockProvider) GetSnapshot(_ context.Context, _ Location) (SnapshotInput, error) {
	return m.fixed, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/weather -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/weather/provider.go backend/internal/weather/provider_test.go
git commit -m "新增天气 Provider 接口与 Mock 实现"
```

---

### Task 3: 用户档案与天气快照存储 + 迁移

**Files:**
- Create: `backend/migrations/003_user_weather.sql`
- Modify: `backend/internal/storage/postgres.go`
- Create: `backend/internal/storage/postgres_user_weather_test.go`

**Step 1: Write the failing test**

```go
package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUserProfileUpsertAndGet(t *testing.T) {
	dsn := os.Getenv("STRIDEWISE_TEST_DSN")
	if dsn == "" {
		t.Skip("STRIDEWISE_TEST_DSN not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer pool.Close()

	store := NewPostgresStore(pool)
	profile := UserProfile{
		UserID:        "u1",
		Gender:        "male",
		Age:           28,
		HeightCM:      175,
		WeightKG:      65,
		GoalType:      "5k",
		GoalCycle:     "8w",
		GoalFrequency: 3,
		GoalPace:      "05:30",
		FitnessLevel:  "beginner",
		LocationLat:   31.2,
		LocationLng:   121.5,
		Country:       "CN",
		Province:      "SH",
		City:          "Shanghai",
		LocationSource:"manual",
	}
	if err := store.UpsertUserProfile(context.Background(), profile); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	got, err := store.GetUserProfile(context.Background(), "u1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.LocationLat != 31.2 {
		t.Fatalf("expected lat 31.2, got %v", got.LocationLat)
	}

	snapshot := WeatherSnapshot{
		UserID:            "u1",
		Date:              time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		TemperatureC:      18,
		FeelsLikeC:        18,
		Humidity:          0.4,
		WindSpeedMS:       2,
		PrecipitationProb: 0.1,
		AQI:               50,
		UVIndex:           2,
		RiskLevel:         "green",
	}
	if err := store.CreateWeatherSnapshot(context.Background(), snapshot); err != nil {
		t.Fatalf("create snapshot failed: %v", err)
	}
	gotSnap, err := store.GetWeatherSnapshot(context.Background(), "u1", snapshot.Date)
	if err != nil {
		t.Fatalf("get snapshot failed: %v", err)
	}
	if gotSnap.RiskLevel != "green" {
		t.Fatalf("expected green, got %s", gotSnap.RiskLevel)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`  
Expected: FAIL (undefined: UserProfile / WeatherSnapshot / methods)

**Step 3: Write minimal implementation**

Create migration `003_user_weather.sql`:

```sql
CREATE TABLE IF NOT EXISTS user_profiles (
  user_id TEXT PRIMARY KEY,
  gender TEXT NOT NULL,
  age INT NOT NULL,
  height_cm INT NOT NULL,
  weight_kg INT NOT NULL,
  goal_type TEXT NOT NULL,
  goal_cycle TEXT NOT NULL,
  goal_frequency INT NOT NULL,
  goal_pace TEXT NOT NULL,
  fitness_level TEXT NOT NULL,
  location_lat DOUBLE PRECISION NOT NULL,
  location_lng DOUBLE PRECISION NOT NULL,
  country TEXT NOT NULL,
  province TEXT NOT NULL,
  city TEXT NOT NULL,
  location_source TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS weather_snapshots (
  snapshot_id BIGSERIAL PRIMARY KEY,
  user_id TEXT NOT NULL,
  date DATE NOT NULL,
  temperature_c DOUBLE PRECISION NOT NULL,
  feels_like_c DOUBLE PRECISION NOT NULL,
  humidity DOUBLE PRECISION NOT NULL,
  wind_speed_ms DOUBLE PRECISION NOT NULL,
  precipitation_prob DOUBLE PRECISION NOT NULL,
  aqi INT NOT NULL,
  uv_index DOUBLE PRECISION NOT NULL,
  risk_level TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, date)
);
```

Update `backend/internal/storage/postgres.go` (add structs + methods):

```go
type UserProfile struct {
	UserID         string
	Gender         string
	Age            int
	HeightCM       int
	WeightKG       int
	GoalType       string
	GoalCycle      string
	GoalFrequency  int
	GoalPace       string
	FitnessLevel   string
	LocationLat    float64
	LocationLng    float64
	Country        string
	Province       string
	City           string
	LocationSource string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type WeatherSnapshot struct {
	UserID            string
	Date              time.Time
	TemperatureC      float64
	FeelsLikeC        float64
	Humidity          float64
	WindSpeedMS       float64
	PrecipitationProb float64
	AQI               int
	UVIndex           float64
	RiskLevel         string
	CreatedAt         time.Time
}

func (s *PostgresStore) UpsertUserProfile(ctx context.Context, p UserProfile) error { /* upsert SQL */ }
func (s *PostgresStore) GetUserProfile(ctx context.Context, userID string) (UserProfile, error) { /* select SQL */ }
func (s *PostgresStore) CreateWeatherSnapshot(ctx context.Context, w WeatherSnapshot) error { /* insert SQL */ }
func (s *PostgresStore) GetWeatherSnapshot(ctx context.Context, userID string, date time.Time) (WeatherSnapshot, error) { /* select SQL */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`  
Expected: PASS (or SKIP if `STRIDEWISE_TEST_DSN` not set)

**Step 5: Commit**

```bash
git add backend/migrations/003_user_weather.sql backend/internal/storage/postgres.go backend/internal/storage/postgres_user_weather_test.go
git commit -m "新增用户档案与天气快照存储"
```

---

### Task 4: 内部接口与校验（User Profile + Weather）

**Files:**
- Modify: `backend/internal/server/http.go`
- Modify: `backend/cmd/api/main.go`
- Create: `backend/internal/server/http_user_weather_test.go`

**Step 1: Write the failing test**

```go
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/weather"
)

type fakeStore struct {
	profile storage.UserProfile
	snap    storage.WeatherSnapshot
}

func (f *fakeStore) UpsertUserProfile(_ context.Context, p storage.UserProfile) error { f.profile = p; return nil }
func (f *fakeStore) GetUserProfile(_ context.Context, _ string) (storage.UserProfile, error) { return f.profile, nil }
func (f *fakeStore) CreateWeatherSnapshot(_ context.Context, s storage.WeatherSnapshot) error { f.snap = s; return nil }
func (f *fakeStore) GetWeatherSnapshot(_ context.Context, _ string, _ time.Time) (storage.WeatherSnapshot, error) { return f.snap, nil }

func TestCreateUserProfile_RequiresLocation(t *testing.T) {
	store := &fakeStore{}
	provider := weather.NewMockProvider(weather.SnapshotInput{TemperatureC: 20})

	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, store, provider)

	body := map[string]any{
		"user_id": "u1",
		"gender": "male",
		"age": 20,
		"height_cm": 175,
		"weight_kg": 65,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/user/profile", bytes.NewReader(b))
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/server -v`  
Expected: FAIL (NewHTTPServer signature mismatch / missing endpoints)

**Step 3: Write minimal implementation**

- 在 `http.go` 增加依赖接口：
  - `type UserProfileStore interface { UpsertUserProfile(...); GetUserProfile(...) }`
  - `type WeatherStore interface { CreateWeatherSnapshot(...); GetWeatherSnapshot(...) }`
  - `type WeatherProvider interface { GetSnapshot(...) }`
- 扩展 `NewHTTPServer` 参数，注入 `UserProfileStore` 与 `WeatherProvider`。
- 新增处理器：
  - `POST /internal/v1/user/profile`
  - `GET /internal/v1/user/profile?user_id=...`
  - `POST /internal/v1/weather/snapshot`
  - `GET /internal/v1/weather/risk?user_id=...&date=...`
- 增加 `validateUserProfileRequest` 校验位置必填与 `location_source` 合法性。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/server -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_user_weather_test.go backend/cmd/api/main.go
git commit -m "新增用户档案与天气内部接口"
```

---

Plan complete and saved to `docs/plans/2026-03-10-user-weather-implementation-plan.md`. Two execution options:

1. Subagent-Driven (this session) - I dispatch fresh subagent per task, review between tasks, fast iteration  
2. Parallel Session (separate) - Open new session with executing-plans, batch execution with checkpoints

Which approach?
