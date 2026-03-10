# Baseline + Training Summary Implementation Plan

## 文档版本
- 当前版本：v0.2.0
- 发布日期：2026-03-10
- 文档状态：可评审

## 变更记录
| 版本号 | 日期 | 变更说明 |
| --- | --- | --- |
| v0.2.0 | 2026-03-10 | 新增 AI Provider 任务与配置；改为 subagent-driven 执行要求。 |

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** 实现基线评估（含历史与当前值）、训练后总结与反馈接口，并在训练记录/同步后触发 `baseline:recalc` 异步任务完成重算与总结覆盖更新。

**Architecture:** 新增 `baseline_current` / `baseline_history` / `training_summaries` / `training_feedbacks` 四张表；`internal/baseline` 提供计算逻辑；`internal/storage` 负责读写；`internal/server` 提供查询与反馈接口；`internal/worker` 新增 `baseline:recalc` 任务处理并写入基线与总结。

**Tech Stack:** Go、Kratos HTTP、PostgreSQL（pgx）、Asynq

**AI Provider:** OpenAI（默认），保留本地模型可插拔实现

---

### Task 1: 基线计算核心逻辑

**Files:**
- Create: `backend/internal/baseline/calc.go`
- Create: `backend/internal/baseline/calc_test.go`

**Step 1: Write the failing test**

```go
package baseline

import "testing"

func TestCalcPaceAverage(t *testing.T) {
	input := []SessionInput{
		{DistanceKM: 5, PaceSecPerKM: 360},
		{DistanceKM: 10, PaceSecPerKM: 330},
	}
	avg := CalcPaceAverage(input)
	if avg != 340 {
		t.Fatalf("expected 340, got %d", avg)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/baseline -v`  
Expected: FAIL (undefined: SessionInput / CalcPaceAverage)

**Step 3: Write minimal implementation**

```go
package baseline

import "math"

type SessionInput struct {
	DurationMin   float64
	DistanceKM    float64
	RPE           *int
	PaceSecPerKM  int
	StartDayIndex int
}

type Metrics struct {
	DataSessions7d     int
	AcuteSRPE          float64
	ChronicSRPE        float64
	ACWRSRPE           float64
	AcuteDistance      float64
	ChronicDistance    float64
	ACWRDistance       float64
	Monotony           float64
	Strain             float64
	PaceAvgSecPerKM    int
	PaceLowSecPerKM    int
	PaceHighSecPerKM   int
	Status             string
}

func CalcPaceAverage(items []SessionInput) int {
	var totalDist float64
	var weighted float64
	for _, s := range items {
		if s.DistanceKM <= 0 || s.PaceSecPerKM <= 0 {
			continue
		}
		totalDist += s.DistanceKM
		weighted += s.DistanceKM * float64(s.PaceSecPerKM)
	}
	if totalDist == 0 {
		return 0
	}
	return int(math.Round(weighted / totalDist))
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/baseline -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/baseline/calc.go backend/internal/baseline/calc_test.go
git commit -m "新增基线计算核心逻辑"
```

---

### Task 2: 基线计算扩展（ACWR/单调性/压力/门槛）

**Files:**
- Modify: `backend/internal/baseline/calc.go`
- Modify: `backend/internal/baseline/calc_test.go`

**Step 1: Write the failing test**

```go
func TestCalcACWRDistance(t *testing.T) {
	items := []SessionInput{
		{DistanceKM: 5, StartDayIndex: 0},
		{DistanceKM: 5, StartDayIndex: 1},
	}
	m := CalcMetrics(items, 2)
	if m.ACWRDistance <= 0 {
		t.Fatalf("expected acwr distance")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/baseline -v`  
Expected: FAIL (undefined: CalcMetrics)

**Step 3: Write minimal implementation**

```go
func CalcMetrics(items []SessionInput, sessions7d int) Metrics {
	// compute acute/chronic for srpe and distance
	// compute monotony and strain
	// set status = insufficient_data if sessions7d < 3
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/baseline -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/baseline/calc.go backend/internal/baseline/calc_test.go
git commit -m "完善基线计算指标"
```

---

### Task 3: 数据表与存储方法

**Files:**
- Create: `backend/migrations/005_baseline_summary.sql`
- Modify: `backend/internal/storage/postgres.go`
- Create: `backend/internal/storage/postgres_baseline_test.go`

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

func TestBaselineStore(t *testing.T) {
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
	current := BaselineCurrent{
		UserID:         "u1",
		ComputedAt:     time.Now().UTC(),
		DataSessions7d: 3,
		Status:         "ok",
	}
	if err := store.UpsertBaselineCurrent(context.Background(), current); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	history := BaselineHistory{
		BaselineID: "b1",
		UserID:     "u1",
		ComputedAt: time.Now().UTC(),
		TriggerType:"training_create",
		TriggerRef: "log-1",
		Status:     "ok",
	}
	if err := store.CreateBaselineHistory(context.Background(), history); err != nil {
		t.Fatalf("history failed: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`  
Expected: FAIL (undefined types / methods)

**Step 3: Write minimal implementation**

Create migration `005_baseline_summary.sql`:

```sql
CREATE TABLE IF NOT EXISTS baseline_current (
  user_id TEXT PRIMARY KEY,
  computed_at TIMESTAMPTZ NOT NULL,
  data_sessions_7d INT NOT NULL,
  acute_load_srpe DOUBLE PRECISION,
  chronic_load_srpe DOUBLE PRECISION,
  acwr_srpe DOUBLE PRECISION,
  acute_load_distance DOUBLE PRECISION,
  chronic_load_distance DOUBLE PRECISION,
  acwr_distance DOUBLE PRECISION,
  monotony DOUBLE PRECISION,
  strain DOUBLE PRECISION,
  pace_avg_sec_per_km INT,
  pace_low_sec_per_km INT,
  pace_high_sec_per_km INT,
  status TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS baseline_history (
  baseline_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  computed_at TIMESTAMPTZ NOT NULL,
  trigger_type TEXT NOT NULL,
  trigger_ref TEXT NOT NULL,
  data_sessions_7d INT NOT NULL,
  acute_load_srpe DOUBLE PRECISION,
  chronic_load_srpe DOUBLE PRECISION,
  acwr_srpe DOUBLE PRECISION,
  acute_load_distance DOUBLE PRECISION,
  chronic_load_distance DOUBLE PRECISION,
  acwr_distance DOUBLE PRECISION,
  monotony DOUBLE PRECISION,
  strain DOUBLE PRECISION,
  pace_avg_sec_per_km INT,
  pace_low_sec_per_km INT,
  pace_high_sec_per_km INT,
  status TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS training_summaries (
  summary_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  log_id TEXT NOT NULL UNIQUE,
  completion_rate TEXT NOT NULL,
  intensity_match TEXT NOT NULL,
  recovery_advice TEXT NOT NULL,
  anomaly_notes TEXT NOT NULL,
  performance_notes TEXT NOT NULL,
  next_suggestion TEXT NOT NULL,
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS training_feedbacks (
  feedback_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  log_id TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Update `backend/internal/storage/postgres.go`:

```go
type BaselineCurrent struct { /* fields */ }
type BaselineHistory struct { /* fields */ }
type TrainingSummary struct { /* fields */ }
type TrainingFeedback struct { /* fields */ }

func (s *PostgresStore) UpsertBaselineCurrent(ctx context.Context, b BaselineCurrent) error
func (s *PostgresStore) CreateBaselineHistory(ctx context.Context, b BaselineHistory) error
func (s *PostgresStore) ListBaselineHistory(ctx context.Context, userID string, from time.Time, to time.Time) ([]BaselineHistory, error)
func (s *PostgresStore) GetBaselineCurrent(ctx context.Context, userID string) (BaselineCurrent, error)

func (s *PostgresStore) UpsertTrainingSummary(ctx context.Context, s2 TrainingSummary) error
func (s *PostgresStore) ListTrainingSummaries(ctx context.Context, userID string, from time.Time, to time.Time) ([]TrainingSummary, error)
func (s *PostgresStore) CreateTrainingFeedback(ctx context.Context, f TrainingFeedback) error
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`  
Expected: PASS (or SKIP if `STRIDEWISE_TEST_DSN` not set)

**Step 5: Commit**

```bash
git add backend/migrations/005_baseline_summary.sql backend/internal/storage/postgres.go backend/internal/storage/postgres_baseline_test.go
git commit -m "新增基线与训练总结存储"
```

---

### Task 4: Baseline Recalc 任务类型与 Worker

**Files:**
- Modify: `backend/internal/task/task.go`
- Create: `backend/internal/task/task_baseline_test.go`
- Modify: `backend/internal/worker/handler.go`
- Modify: `backend/cmd/worker/main.go`
- Create: `backend/internal/baseline/processor.go`
- Create: `backend/internal/baseline/processor_test.go`

**Step 1: Write the failing test**

```go
package task

import "testing"

func TestEncodeDecodeBaselineRecalcPayload(t *testing.T) {
	p := BaselineRecalcPayload{JobID: "job-1", UserID: "u1", TriggerType: "training_create", TriggerRef: "log-1"}
	b, err := EncodeBaselineRecalcPayload(p)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	out, err := DecodeBaselineRecalcPayload(b)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.JobID != "job-1" {
		t.Fatalf("unexpected payload")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/task -v`  
Expected: FAIL (undefined: BaselineRecalcPayload)

**Step 3: Write minimal implementation**

```go
const TypeBaselineRecalc = "baseline:recalc"

type BaselineRecalcPayload struct {
	JobID       string `json:"job_id"`
	UserID      string `json:"user_id"`
	TriggerType string `json:"trigger_type"`
	TriggerRef  string `json:"trigger_ref"`
}
```

Add handler in worker:

```go
func HandleBaselineRecalc(ctx context.Context, t *asynq.Task) error { /* decode + processor.ProcessBaselineRecalc */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/task -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/task/task.go backend/internal/task/task_baseline_test.go backend/internal/worker/handler.go backend/cmd/worker/main.go backend/internal/baseline/processor.go backend/internal/baseline/processor_test.go
git commit -m "新增基线重算任务类型与处理器"
```

---

### Task 5: 基线与总结接口 + 触发

**Files:**
- Modify: `backend/internal/server/http.go`
- Create: `backend/internal/server/http_baseline_test.go`
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
	"time"

	"stridewise/backend/internal/storage"
)

type baselineStoreStub struct{}

func (b *baselineStoreStub) GetBaselineCurrent(_ context.Context, _ string) (storage.BaselineCurrent, error) {
	return storage.BaselineCurrent{UserID: "u1", Status: "ok"}, nil
}

func TestGetBaselineCurrent(t *testing.T) {
	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_ = b
	_ = srv
	_ = bytes.NewBufferString("")
	_ = json.NewEncoder
	_ = time.Now
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/server -v`  
Expected: FAIL (signature mismatch / missing endpoint)

**Step 3: Write minimal implementation**

新增接口：
- `GET /internal/v1/baseline/current`
- `GET /internal/v1/baseline/history`
- `GET /internal/v1/training/summaries`
- `POST /internal/v1/training/feedback`

并在训练记录创建/更新/删除、第三方同步完成时投递 `baseline:recalc` 任务。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/server -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_baseline_test.go backend/cmd/api/main.go
git commit -m "新增基线与训练总结接口"
```

---

Plan complete and saved to `docs/plans/2026-03-10-baseline-summary-implementation-plan.md`. Two execution options:

1. Subagent-Driven (this session) - I dispatch fresh subagent per task, review between tasks, fast iteration  
2. Parallel Session (separate) - Open new session with executing-plans, batch execution with checkpoints

Which approach?

---

### Task 6: AI Provider 与训练总结生成

**Files:**
- Create: `backend/internal/ai/summarizer.go`
- Create: `backend/internal/ai/openai.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/config/config.yaml`
- Modify: `backend/internal/baseline/processor.go`
- Create: `backend/internal/ai/openai_test.go`

**Step 1: Write the failing test**

```go
package ai

import "testing"

func TestOpenAISummarizerBuildsRequest(t *testing.T) {
	_ = NewOpenAISummarizer(OpenAIConfig{})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/ai -v`
Expected: FAIL (package not found)

**Step 3: Write minimal implementation**

```go
package ai

type Summarizer interface {
	Summarize(input SummaryInput) (SummaryOutput, error)
}

type SummaryInput struct { /* minimal fields */ }
type SummaryOutput struct { /* fields map to training_summaries */ }

type OpenAIConfig struct {
	APIKey    string
	BaseURL   string
	Model     string
	TimeoutMs int
	MaxTokens int
	Temperature float64
}

func NewOpenAISummarizer(cfg OpenAIConfig) *OpenAISummarizer { return &OpenAISummarizer{cfg: cfg} }
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/ai -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/ai/summarizer.go backend/internal/ai/openai.go backend/internal/ai/openai_test.go backend/internal/config/config.go backend/config/config.yaml backend/internal/baseline/processor.go
git commit -m "新增训练总结 AI Provider"
```
