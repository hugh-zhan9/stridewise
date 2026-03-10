# Recommendation Engine Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现“点击即生成”的建议引擎，AI 优先 + 规则兜底，并提供推荐记录与反馈接口。

**Architecture:** 新增 `recommendations`/`recommendation_feedbacks` 表；`internal/recommendation` 负责编排（天气→AI→规则→落库）；`internal/ai` 新增 Recommender 接口与 OpenAI 实现；HTTP 提供生成/查询/反馈接口。

**Tech Stack:** Go、Kratos HTTP、PostgreSQL（pgx）、OpenAI

---

### Task 1: Recommendation 存储与迁移

**Files:**
- Create: `backend/migrations/006_recommendations.sql`
- Modify: `backend/internal/storage/postgres.go`
- Create: `backend/internal/storage/postgres_recommendation_test.go`

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

func TestRecommendationStore(t *testing.T) {
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
	rec := Recommendation{
		RecID:              "r1",
		UserID:             "u1",
		RecommendationDate: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		InputJSON:          []byte(`{"a":1}`),
		OutputJSON:         []byte(`{"b":2}`),
		RiskLevel:          "green",
		OverrideJSON:       []byte(`{}`),
		IsFallback:         false,
		AIProvider:         "openai",
		AIModel:            "gpt-4.1-mini",
		PromptVersion:      "v1",
		EngineVersion:      "v1",
	}
	if err := store.CreateRecommendation(context.Background(), rec); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := store.GetLatestRecommendation(context.Background(), "u1"); err != nil {
		t.Fatalf("get latest failed: %v", err)
	}
	feedback := RecommendationFeedback{
		FeedbackID: "f1",
		RecID:      "r1",
		UserID:     "u1",
		Useful:     "yes",
		Reason:     "ok",
	}
	if err := store.CreateRecommendationFeedback(context.Background(), feedback); err != nil {
		t.Fatalf("feedback failed: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`  
Expected: FAIL (undefined types / methods)

**Step 3: Write minimal implementation**

Create migration `006_recommendations.sql`:

```sql
CREATE TABLE IF NOT EXISTS recommendations (
  rec_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  recommendation_date DATE NOT NULL,
  input_json JSONB NOT NULL,
  output_json JSONB NOT NULL,
  risk_level TEXT NOT NULL,
  override_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  is_fallback BOOLEAN NOT NULL DEFAULT FALSE,
  ai_provider TEXT NOT NULL,
  ai_model TEXT NOT NULL,
  prompt_version TEXT NOT NULL,
  engine_version TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recommendations_user_time ON recommendations(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS recommendation_feedbacks (
  feedback_id TEXT PRIMARY KEY,
  rec_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  useful TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (rec_id, user_id)
);
```

Update `postgres.go`:
- `type Recommendation`, `type RecommendationFeedback`
- `CreateRecommendation(ctx, rec)`
- `GetLatestRecommendation(ctx, userID)`
- `CreateRecommendationFeedback(ctx, feedback)`

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`  
Expected: PASS (or SKIP if `STRIDEWISE_TEST_DSN` not set)

**Step 5: Commit**

```bash
git add backend/migrations/006_recommendations.sql backend/internal/storage/postgres.go backend/internal/storage/postgres_recommendation_test.go
git commit -m "新增建议记录与反馈存储"
```

---

### Task 2: AI Recommender 接口与 OpenAI 实现

**Files:**
- Create: `backend/internal/ai/recommender.go`
- Create: `backend/internal/ai/openai_recommender.go`
- Create: `backend/internal/ai/openai_recommender_test.go`

**Step 1: Write the failing test**

```go
package ai

import (
	"context"
	"testing"
)

func TestOpenAIRecommenderRequiresConfig(t *testing.T) {
	r := NewOpenAIRecommender(OpenAIConfig{})
	_, err := r.Recommend(context.Background(), RecommendationInput{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/ai -v`  
Expected: FAIL (undefined: NewOpenAIRecommender)

**Step 3: Write minimal implementation**

```go
type Recommender interface {
	Recommend(ctx context.Context, input RecommendationInput) (RecommendationOutput, error)
}

type RecommendationInput struct { /* user + baseline + weather */ }
type RecommendationOutput struct { /* structured fields */ }
```

Implement OpenAI recommender using Chat Completions JSON mode，复用 `OpenAIConfig`。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/ai -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/ai/recommender.go backend/internal/ai/openai_recommender.go backend/internal/ai/openai_recommender_test.go
git commit -m "新增AI建议生成接口"
```

---

### Task 3: 规则兜底与覆盖逻辑

**Files:**
- Create: `backend/internal/recommendation/rules.go`
- Create: `backend/internal/recommendation/rules_test.go`

**Step 1: Write the failing test**

```go
package recommendation

import "testing"

func TestApplyRules_RedWeatherOverrides(t *testing.T) {
	input := RuleInput{WeatherRisk: "red"}
	out := RecommendationOutput{ShouldRun: true}
	result := ApplyRules(input, out)
	if result.Output.ShouldRun {
		t.Fatalf("expected override to rest")
	}
	if result.OverrideReason == "" {
		t.Fatalf("expected override reason")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`  
Expected: FAIL (undefined: ApplyRules)

**Step 3: Write minimal implementation**

```go
type RuleInput struct { WeatherRisk string; HasDiscomfort bool; HighLoad bool }
type RuleResult struct { Output RecommendationOutput; OverrideReason string }
```

Rules: red 风险/不适/高负荷 → 强制休息或降级。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/recommendation/rules.go backend/internal/recommendation/rules_test.go
git commit -m "新增建议规则兜底逻辑"
```

---

### Task 4: Recommendation 编排器（天气→AI→规则→落库）

**Files:**
- Create: `backend/internal/recommendation/processor.go`
- Create: `backend/internal/recommendation/processor_test.go`
- Modify: `backend/internal/storage/postgres.go` (如需查询 latest weather / latest training)

**Step 1: Write the failing test**

```go
package recommendation

import (
	"context"
	"testing"
	"time"

	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/weather"
)

type fakeStore struct { created bool }

func (f *fakeStore) CreateRecommendation(_ context.Context, _ storage.Recommendation) error { f.created = true; return nil }
func (f *fakeStore) GetLatestRecommendation(_ context.Context, _ string) (storage.Recommendation, error) { return storage.Recommendation{}, nil }
func (f *fakeStore) GetUserProfile(_ context.Context, _ string) (storage.UserProfile, error) { return storage.UserProfile{UserID: "u1", LocationLat: 1, LocationLng: 2, Country: "CN", Province: "SH", City: "SH"}, nil }
func (f *fakeStore) GetBaselineCurrent(_ context.Context, _ string) (storage.BaselineCurrent, error) { return storage.BaselineCurrent{UserID: "u1"}, nil }
func (f *fakeStore) CreateWeatherSnapshot(_ context.Context, _ storage.WeatherSnapshot) error { return nil }
func (f *fakeStore) GetLatestWeatherSnapshot(_ context.Context, _ string) (storage.WeatherSnapshot, error) { return storage.WeatherSnapshot{}, nil }

type fakeAI struct{}
func (fakeAI) Recommend(_ context.Context, _ RecommendationInput) (RecommendationOutput, error) {
	return RecommendationOutput{ShouldRun: true, Explanation: []string{"a", "b"}}, nil
}

type fakeWeather struct{}
func (fakeWeather) GetSnapshot(_ context.Context, _ weather.Location) (weather.SnapshotInput, error) {
	return weather.SnapshotInput{TemperatureC: 20}, nil
}

func TestGenerateRecommendation(t *testing.T) {
	store := &fakeStore{}
	p := NewProcessor(store, fakeWeather{}, fakeAI{})
	p.now = func() time.Time { return time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC) }
	if _, err := p.Generate(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !store.created {
		t.Fatalf("expected create recommendation")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`  
Expected: FAIL (undefined processor)

**Step 3: Write minimal implementation**

Processor responsibilities:
- 拉最新天气，失败则使用最近快照
- 读取用户档案/基线/近期不适
- 生成 AI 输入，调用 AI
- 规则兜底校验，必要时覆盖
- 落库 Recommendation

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/recommendation/processor.go backend/internal/recommendation/processor_test.go
git commit -m "新增建议生成编排器"
```

---

### Task 5: HTTP 接口（生成/最新/反馈）

**Files:**
- Modify: `backend/internal/server/http.go`
- Create: `backend/internal/server/http_recommendation_test.go`
- Modify: `backend/cmd/api/main.go`

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

	"stridewise/backend/internal/storage"
)

type recServiceStub struct{}

func (recServiceStub) Generate(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{RecID: "r1"}, nil
}
func (recServiceStub) GetLatest(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{RecID: "r1"}, nil
}
func (recServiceStub) Feedback(_ context.Context, _ string, _ string, _ string, _ string) error {
	return nil
}

func TestGenerateRecommendation(t *testing.T) {
	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, recServiceStub{})
	body := map[string]any{"user_id": "u1"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/recommendations/generate", bytes.NewReader(b))
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/server -v`  
Expected: FAIL (missing routes / signature)

**Step 3: Write minimal implementation**

新增：
- `POST /internal/v1/recommendations/generate`
- `GET /internal/v1/recommendations/latest`
- `POST /internal/v1/recommendations/{id}/feedback`

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/server -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_recommendation_test.go backend/cmd/api/main.go
git commit -m "新增建议接口"
```

---

### Task 6: 配置与注入

**Files:**
- Modify: `backend/cmd/api/main.go`

**Step 1: Write the failing test**

```go
// 通过编译失败驱动依赖注入
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/cmd/api -v`  
Expected: FAIL (missing dependency wiring)

**Step 3: Write minimal implementation**

在 API 启动时构造 `recommendation.Processor`，注入：
- `weather.Provider`
- `ai.Recommender`
- `storage.PostgresStore`

**Step 4: Run test to verify it passes**

Run: `go test ./backend/cmd/api -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/api/main.go
git commit -m "接入建议引擎依赖"
```

---

Plan complete and saved to `docs/plans/2026-03-10-recommendation-implementation-plan.md`. Two execution options:

1. Subagent-Driven (this session) - I dispatch fresh subagent per task, review between tasks, fast iteration  
2. Parallel Session (separate) - Open new session with executing-plans, batch execution with checkpoints

Which approach?
