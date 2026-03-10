# 训练总结/反馈支持第三方活动 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 训练总结与训练反馈支持 `log/activity` 两类来源，支持第三方导入活动生成总结，并在训练删除时软删总结与反馈。

**Architecture:** 扩展 `training_summaries`/`training_feedbacks` 表结构引入 `source_type/source_id`，调整存储与查询；在基线重算任务内补齐活动总结；HTTP 训练反馈改为 `source_type/source_id` 入参；删除训练记录时软删对应总结与反馈。

**Tech Stack:** Go、Kratos HTTP、PostgreSQL、Asynq、OpenAI

---

### Task 1: 迁移与存储模型扩展（summary/feedback 支持 source）

**Files:**
- Create: `backend/migrations/007_training_summary_activity.sql`
- Modify: `backend/internal/storage/postgres.go`
- Modify: `backend/internal/storage/postgres_baseline_test.go`

**Step 1: Write the failing test**

Update `TestGetTrainingSummary`:

```go
summary := TrainingSummary{
    SummaryID:        "s1",
    UserID:           "u1",
    SourceType:       "log",
    SourceID:         "log-1",
    LogID:            "log-1",
    CompletionRate:   "ok",
    IntensityMatch:   "ok",
    RecoveryAdvice:   "ok",
    AnomalyNotes:     "ok",
    PerformanceNotes: "ok",
    NextSuggestion:   "ok",
}
if err := store.UpsertTrainingSummary(context.Background(), summary); err != nil {
    t.Fatalf("upsert summary failed: %v", err)
}
if _, err := store.GetTrainingSummaryBySource(context.Background(), "log", "log-1"); err != nil {
    t.Fatalf("get summary failed: %v", err)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`
Expected: FAIL (missing fields/methods)

**Step 3: Write minimal implementation**

Migration `007_training_summary_activity.sql`:
- `training_summaries`:
  - `ADD COLUMN source_type TEXT NOT NULL DEFAULT 'log'`
  - `ADD COLUMN source_id TEXT`
  - `UPDATE training_summaries SET source_id = log_id WHERE source_id IS NULL`
  - `ALTER COLUMN source_id SET NOT NULL`
  - `ALTER COLUMN log_id DROP NOT NULL`
  - `DROP CONSTRAINT training_summaries_log_id_key`
  - `ADD UNIQUE (user_id, source_type, source_id)`
- `training_feedbacks`:
  - `ADD COLUMN source_type TEXT NOT NULL DEFAULT 'log'`
  - `ADD COLUMN source_id TEXT`
  - `UPDATE training_feedbacks SET source_id = log_id WHERE source_id IS NULL`
  - `ALTER COLUMN source_id SET NOT NULL`
  - `ALTER COLUMN log_id DROP NOT NULL`
  - `ADD COLUMN deleted_at TIMESTAMPTZ`
  - `ADD UNIQUE (user_id, source_type, source_id)`

Update `storage` models & methods:
- `TrainingSummary` add `SourceType`, `SourceID`
- `TrainingFeedback` add `SourceType`, `SourceID`, `DeletedAt`
- Update `UpsertTrainingSummary` to write `source_type/source_id`
- Add `GetTrainingSummaryBySource(ctx, sourceType, sourceID)`
- Update `CreateTrainingFeedback` to write `source_type/source_id` and optional `log_id`
- Add `SoftDeleteTrainingSummaryBySource` + `SoftDeleteTrainingFeedbackBySource`
- Extend `Activity` with `ID`, `Source`, `SourceActivityID`, `Name`
- Add `ListActivitiesBySyncJob(ctx, jobID string)`

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`
Expected: PASS (or SKIP if `STRIDEWISE_TEST_DSN` not set)

**Step 5: Commit**

```bash
git add backend/migrations/007_training_summary_activity.sql backend/internal/storage/postgres.go backend/internal/storage/postgres_baseline_test.go
git commit -m "扩展训练总结与反馈存储"
```

---

### Task 2: 基线重算支持活动总结与删除软删

**Files:**
- Modify: `backend/internal/baseline/processor.go`
- Modify: `backend/internal/baseline/processor_test.go`

**Step 1: Write the failing test**

```go
func TestBaselineSyncGeneratesActivitySummaries(t *testing.T) {
    store := &fakeBaselineStore{activitiesByJob: []storage.Activity{{
        ID: 1,
        UserID: "u1",
        Source: "keep",
        SourceActivityID: "a1",
        Name: "activity",
        DistanceM: 5000,
        MovingTimeSec: 1500,
        StartTimeLocal: time.Now(),
    }}}
    p := NewProcessor(store)
    p.SetSummarizer(fakeSummarizer{})

    _, err := p.recalc(context.Background(), "u1", "sync", "job-1")
    if err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
    if store.activitySummaryCreated == false {
        t.Fatalf("expected activity summary created")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/baseline -v`
Expected: FAIL (missing store methods / logic)

**Step 3: Write minimal implementation**

- Extend `Store` interface:
  - `ListActivitiesBySyncJob(ctx, jobID string) ([]storage.Activity, error)`
  - `GetTrainingSummaryBySource(ctx, sourceType, sourceID string)`
  - `SoftDeleteTrainingSummaryBySource(ctx, sourceType, sourceID string)`
  - `SoftDeleteTrainingFeedbackBySource(ctx, sourceType, sourceID string)`
- For `triggerType == "sync"`:
  - Load activities by `jobID`
  - For each activity, generate summary with `source_type=activity`
  - If summarizer fails, fallback only when summary does not exist
- For `triggerType == "training_delete"`:
  - Soft delete summary + feedback for `source_type=log`, `source_id=log_id`
- For `training_create/update`:
  - Use `source_type=log`, `source_id=log_id` in summary storage

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/baseline -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/baseline/processor.go backend/internal/baseline/processor_test.go
git commit -m "支持活动训练总结与删除软删"
```

---

### Task 3: 训练总结/反馈 API 支持 source 字段

**Files:**
- Modify: `backend/internal/server/http.go`
- Modify: `backend/internal/server/http_baseline_test.go`

**Step 1: Write the failing test**

Add new test in `http_baseline_test.go`:

```go
func TestCreateTrainingFeedbackWithSource(t *testing.T) {
    srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, &baselineStoreStub{}, nil)

    body := `{"user_id":"u1","source_type":"activity","source_id":"1","content":"ok"}`
    req := httptest.NewRequest(http.MethodPost, "/internal/v1/training/feedback", strings.NewReader(body))
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
Expected: FAIL (missing fields/validation)

**Step 3: Write minimal implementation**

- Update `trainingFeedbackRequest` to include `source_type/source_id`
- Validate `source_type` in `{log, activity}` and require `source_id`
- Map `log_id` when `source_type=log`
- Ensure feedback creation uses new fields

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/server -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_baseline_test.go
git commit -m "训练反馈接口支持source"
```

---

### Task 4: 训练总结查询支持活动（排序统一）

**Files:**
- Modify: `backend/internal/storage/postgres.go`
- Modify: `backend/internal/storage/postgres_baseline_test.go`

**Step 1: Write the failing test**

Add test in `postgres_baseline_test.go`:

```go
func TestListTrainingSummariesIncludesActivity(t *testing.T) {
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
    var activityID int64
    err = pool.QueryRow(context.Background(), `
        INSERT INTO activities (user_id, source, source_activity_id, name, distance_m, moving_time_sec, start_time_utc, start_time_local, timezone)
        VALUES ('u1','keep','a1','act',5000,1500,NOW(),NOW(),'UTC')
        RETURNING id
    `).Scan(&activityID)
    if err != nil {
        t.Fatalf("insert activity failed: %v", err)
    }

    summary := TrainingSummary{
        SummaryID:        "s-act",
        UserID:           "u1",
        SourceType:       "activity",
        SourceID:         fmt.Sprintf("%d", activityID),
        CompletionRate:   "ok",
        IntensityMatch:   "ok",
        RecoveryAdvice:   "ok",
        AnomalyNotes:     "ok",
        PerformanceNotes: "ok",
        NextSuggestion:   "ok",
    }
    if err := store.UpsertTrainingSummary(context.Background(), summary); err != nil {
        t.Fatalf("upsert summary failed: %v", err)
    }

    from := time.Now().Add(-24 * time.Hour)
    to := time.Now().Add(24 * time.Hour)
    summaries, err := store.ListTrainingSummaries(context.Background(), "u1", from, to)
    if err != nil {
        t.Fatalf("list summaries failed: %v", err)
    }
    if len(summaries) == 0 {
        t.Fatalf("expected activity summary in list")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`
Expected: FAIL (missing fields/logic)

**Step 3: Write minimal implementation**

- `ListTrainingSummaries` 使用 UNION 查询 log + activity
- 返回结果包含 `source_type/source_id`
- 排序字段统一为 `start_time` / `start_time_local`

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/storage/postgres.go backend/internal/storage/postgres_baseline_test.go
git commit -m "训练总结查询支持活动"
```

---

### Task 5: 全量回归与变更记录

**Files:**
- Modify: `docs/AI_CHANGELOG.md`

**Step 1: Run full tests**

Run: `go test ./backend/...`
Expected: PASS (storage tests may SKIP without DSN)

**Step 2: Flight Recorder**

Run:
```bash
python3 /Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py "Feature" "训练总结/反馈支持活动来源与软删除" "新增source字段与同步活动总结，若迁移或查询逻辑有误会影响历史数据展示" "S2"
```

**Step 3: Commit**

```bash
git add docs/AI_CHANGELOG.md
git commit -m "记录训练总结支持活动变更"
```
