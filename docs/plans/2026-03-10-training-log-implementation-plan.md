# Training Log (Manual) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 新增手动训练记录的存储与接口，支持更新/软删并触发异步重算任务，满足冲突拒绝与校验规则。

**Architecture:** 新增 `training_logs` 与 `async_jobs` 两张表；`internal/training` 提供解析与枚举规范化；`internal/storage` 负责训练记录与异步任务状态；`internal/server` 提供训练记录 CRUD 接口并投递 `training_recalc` 异步任务；`internal/worker` 处理重算任务并更新状态。

**Tech Stack:** Go、Kratos HTTP、PostgreSQL（pgx）、Asynq

---

### Task 1: 训练记录解析与类型规范化

**Files:**
- Create: `backend/internal/training/parse.go`
- Create: `backend/internal/training/parse_test.go`

**Step 1: Write the failing test**

```go
package training

import "testing"

func TestParseDuration(t *testing.T) {
	sec, err := ParseDuration("01:02:03")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sec != 3723 {
		t.Fatalf("expected 3723, got %d", sec)
	}
}

func TestParsePace(t *testing.T) {
	sec, err := ParsePace("05'30''")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sec != 330 {
		t.Fatalf("expected 330, got %d", sec)
	}
}

func TestNormalizeTrainingType_Custom(t *testing.T) {
	tp, custom, err := NormalizeTrainingType("自由跑")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if tp != "custom" || custom != "自由跑" {
		t.Fatalf("unexpected result: %s %s", tp, custom)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/training -v`  
Expected: FAIL (undefined: ParseDuration / ParsePace / NormalizeTrainingType)

**Step 3: Write minimal implementation**

```go
package training

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var allowedTypes = map[string]struct{}{
	"轻松跑": {},
	"有氧跑": {},
	"间歇跑": {},
	"长距离": {},
}

func ParseDuration(input string) (int, error) {
	parts := strings.Split(input, ":")
	if len(parts) != 3 {
		return 0, errors.New("duration format must be HH:MM:SS")
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 {
		return 0, errors.New("duration hours invalid")
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m >= 60 {
		return 0, errors.New("duration minutes invalid")
	}
	s, err := strconv.Atoi(parts[2])
	if err != nil || s < 0 || s >= 60 {
		return 0, errors.New("duration seconds invalid")
	}
	return h*3600 + m*60 + s, nil
}

func ParsePace(input string) (int, error) {
	trimmed := strings.TrimSpace(input)
	trimmed = strings.ReplaceAll(trimmed, "''", "")
	trimmed = strings.ReplaceAll(trimmed, "'", "")
	parts := strings.Split(trimmed, ":")
	if len(parts) != 2 {
		return 0, errors.New("pace format must be mm'ss''")
	}
	m, err := strconv.Atoi(parts[0])
	if err != nil || m <= 0 {
		return 0, errors.New("pace minutes invalid")
	}
	s, err := strconv.Atoi(parts[1])
	if err != nil || s < 0 || s >= 60 {
		return 0, errors.New("pace seconds invalid")
	}
	return m*60 + s, nil
}

func NormalizeTrainingType(input string) (string, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", errors.New("training_type required")
	}
	if _, ok := allowedTypes[input]; ok {
		return input, "", nil
	}
	return "custom", input, nil
}

func FormatPace(sec int) string {
	if sec <= 0 {
		return ""
	}
	min := sec / 60
	sec = sec % 60
	return fmt.Sprintf("%02d'%02d''", min, sec)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/training -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/training/parse.go backend/internal/training/parse_test.go
git commit -m "新增训练记录解析与类型规范化"
```

---

### Task 2: 训练记录与异步任务数据表 + 存储方法

**Files:**
- Create: `backend/migrations/004_training_logs.sql`
- Modify: `backend/internal/storage/postgres.go`
- Create: `backend/internal/storage/postgres_training_test.go`

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

func TestTrainingLogCRUD(t *testing.T) {
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
	start := time.Date(2026, 3, 10, 7, 0, 0, 0, time.UTC)

	log := TrainingLog{
		LogID:          "log-1",
		UserID:         "u1",
		Source:         "manual",
		TrainingType:   "轻松跑",
		StartTime:      start,
		DurationSec:    3600,
		DistanceKM:     10.5,
		PaceStr:        "05'30''",
		PaceSecPerKM:   330,
		RPE:            6,
		Discomfort:     false,
	}
	if err := store.CreateTrainingLog(context.Background(), log); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	log.DistanceKM = 11.0
	if err := store.UpdateTrainingLog(context.Background(), log); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if err := store.SoftDeleteTrainingLog(context.Background(), log.LogID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`  
Expected: FAIL (undefined: TrainingLog / CreateTrainingLog / UpdateTrainingLog / SoftDeleteTrainingLog)

**Step 3: Write minimal implementation**

Create migration `004_training_logs.sql`:

```sql
CREATE TABLE IF NOT EXISTS training_logs (
  log_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  source TEXT NOT NULL,
  training_type TEXT NOT NULL,
  training_type_custom TEXT NOT NULL DEFAULT '',
  start_time TIMESTAMP NOT NULL,
  duration_sec INT NOT NULL,
  distance_km NUMERIC NOT NULL,
  pace_str TEXT NOT NULL,
  pace_sec_per_km INT NOT NULL,
  rpe INT NOT NULL,
  discomfort BOOLEAN NOT NULL,
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS async_jobs (
  job_id TEXT PRIMARY KEY,
  job_type TEXT NOT NULL,
  user_id TEXT NOT NULL,
  payload_json JSONB NOT NULL,
  status TEXT NOT NULL,
  retry_count INT NOT NULL,
  error_message TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Update `backend/internal/storage/postgres.go`:

```go
type TrainingLog struct {
	LogID              string
	UserID             string
	Source             string
	TrainingType       string
	TrainingTypeCustom string
	StartTime          time.Time
	DurationSec        int
	DistanceKM         float64
	PaceStr            string
	PaceSecPerKM       int
	RPE                int
	Discomfort         bool
	DeletedAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type AsyncJob struct {
	JobID        string
	JobType      string
	UserID       string
	PayloadJSON  []byte
	Status       string
	RetryCount   int
	ErrorMessage string
}

func (s *PostgresStore) CreateTrainingLog(ctx context.Context, log TrainingLog) error { /* insert */ }
func (s *PostgresStore) UpdateTrainingLog(ctx context.Context, log TrainingLog) error { /* update */ }
func (s *PostgresStore) SoftDeleteTrainingLog(ctx context.Context, logID string) error { /* update deleted_at */ }
func (s *PostgresStore) CreateAsyncJob(ctx context.Context, job AsyncJob) error { /* insert */ }
func (s *PostgresStore) UpdateAsyncJobStatus(ctx context.Context, jobID, status string, retryCount int, errMsg string) error { /* update */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`  
Expected: PASS (or SKIP if `STRIDEWISE_TEST_DSN` not set)

**Step 5: Commit**

```bash
git add backend/migrations/004_training_logs.sql backend/internal/storage/postgres.go backend/internal/storage/postgres_training_test.go
git commit -m "新增训练记录与异步任务存储"
```

---

### Task 3: 冲突检测存储方法

**Files:**
- Modify: `backend/internal/storage/postgres.go`
- Modify: `backend/internal/storage/postgres_training_test.go`

**Step 1: Write the failing test**

```go
func TestTrainingLogConflict(t *testing.T) {
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
	start := time.Date(2026, 3, 10, 7, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute)
	conflict, err := store.HasTrainingConflict(context.Background(), "u1", start, end)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	_ = conflict
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`  
Expected: FAIL (undefined: HasTrainingConflict)

**Step 3: Write minimal implementation**

```go
func (s *PostgresStore) HasTrainingConflict(ctx context.Context, userID string, start time.Time, end time.Time) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM training_logs
			WHERE user_id=$1 AND deleted_at IS NULL
			  AND (start_time, start_time + (duration_sec || ' seconds')::interval) OVERLAPS ($2, $3)
			UNION ALL
			SELECT 1 FROM activities
			WHERE user_id=$1
			  AND (start_time_local, start_time_local + (moving_time_sec || ' seconds')::interval) OVERLAPS ($2, $3)
			LIMIT 1
		)
	`, userID, start, end).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`  
Expected: PASS (or SKIP if `STRIDEWISE_TEST_DSN` not set)

**Step 5: Commit**

```bash
git add backend/internal/storage/postgres.go backend/internal/storage/postgres_training_test.go
git commit -m "新增训练记录冲突检测"
```

---

### Task 4: 异步任务类型与 Worker 处理

**Files:**
- Modify: `backend/internal/task/task.go`
- Create: `backend/internal/task/task_training_test.go`
- Modify: `backend/internal/worker/handler.go`
- Modify: `backend/cmd/worker/main.go`

**Step 1: Write the failing test**

```go
package task

import "testing"

func TestEncodeDecodeTrainingRecalcPayload(t *testing.T) {
	p := TrainingRecalcPayload{JobID: "job-1", UserID: "u1", LogID: "log-1", Operation: "create"}
	b, err := EncodeTrainingRecalcPayload(p)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	out, err := DecodeTrainingRecalcPayload(b)
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
Expected: FAIL (undefined: TrainingRecalcPayload / EncodeTrainingRecalcPayload)

**Step 3: Write minimal implementation**

```go
const TypeTrainingRecalc = "training:recalc"

type TrainingRecalcPayload struct {
	JobID     string `json:"job_id"`
	UserID    string `json:"user_id"`
	LogID     string `json:"log_id"`
	Operation string `json:"operation"` // create/update/delete
}

func EncodeTrainingRecalcPayload(p TrainingRecalcPayload) ([]byte, error) { /* validate + json */ }
func DecodeTrainingRecalcPayload(b []byte) (TrainingRecalcPayload, error) { /* validate */ }
```

Update `worker/handler.go`:

```go
func HandleTrainingRecalc(ctx context.Context, t *asynq.Task) error {
	p, err := task.DecodeTrainingRecalcPayload(t.Payload())
	if err != nil {
		return err
	}
	return processor.ProcessTrainingRecalc(ctx, p.JobID, p.UserID, p.LogID, p.Operation)
}
```

Update `cmd/worker/main.go` to register handler:

```go
mux.HandleFunc(task.TypeTrainingRecalc, worker.HandleTrainingRecalc)
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/task -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/task/task.go backend/internal/task/task_training_test.go backend/internal/worker/handler.go backend/cmd/worker/main.go
git commit -m "新增训练重算任务类型与处理器"
```

---

### Task 5: 训练记录接口与异步投递

**Files:**
- Modify: `backend/internal/server/http.go`
- Create: `backend/internal/server/http_training_test.go`
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
	"stridewise/backend/internal/task"
)

type trainingStoreStub struct {
	conflict bool
}

func (s *trainingStoreStub) HasTrainingConflict(_ context.Context, _ string, _ time.Time, _ time.Time) (bool, error) {
	return s.conflict, nil
}

func (s *trainingStoreStub) CreateTrainingLog(_ context.Context, _ storage.TrainingLog) error { return nil }
func (s *trainingStoreStub) UpdateTrainingLog(_ context.Context, _ storage.TrainingLog) error { return nil }
func (s *trainingStoreStub) SoftDeleteTrainingLog(_ context.Context, _ string) error { return nil }
func (s *trainingStoreStub) ListTrainingLogs(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.TrainingLog, error) {
	return nil, nil
}

type asyncJobStub struct {
	job storage.AsyncJob
}

func (a *asyncJobStub) CreateAsyncJob(_ context.Context, job storage.AsyncJob) error {
	a.job = job
	return nil
}

func TestCreateTrainingLog_Conflict(t *testing.T) {
	store := &trainingStoreStub{conflict: true}
	jobStore := &asyncJobStub{}
	server := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, store, jobStore)

	body := map[string]any{
		"user_id": "u1",
		"training_type": "轻松跑",
		"start_time": "2026-03-10 07:00:00",
		"duration": "00:30:00",
		"distance_km": 5.0,
		"pace": "06'00''",
		"rpe": 5,
		"discomfort": false,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/training/logs", bytes.NewReader(b))
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
	if jobStore.job.JobID != "" {
		t.Fatalf("job should not be created on conflict")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/server -v`  
Expected: FAIL (NewHTTPServer signature mismatch / missing endpoints)

**Step 3: Write minimal implementation**

- 扩展 `NewHTTPServer` 注入：
  - `TrainingLogStore`
  - `AsyncJobStore`
- 新增请求/响应结构体
- 实现新增/更新/删除/查询逻辑
- 解析 `start_time`/`duration`/`pace`，调用 `NormalizeTrainingType`
- 冲突时返回 `409`
- 成功后创建 `async_jobs` 记录并投递 `training:recalc` 任务

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/server -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_training_test.go backend/cmd/api/main.go
git commit -m "新增训练记录接口与异步重算投递"
```

---

Plan complete and saved to `docs/plans/2026-03-10-training-log-implementation-plan.md`. Two execution options:

1. Subagent-Driven (this session) - I dispatch fresh subagent per task, review between tasks, fast iteration  
2. Parallel Session (separate) - Open new session with executing-plans, batch execution with checkpoints

Which approach?
