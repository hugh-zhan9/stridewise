# 夜间基线重算 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 每日 02:00（服务器时区）触发基线重算，仅覆盖近 28 天有训练/活动的用户。

**Architecture:** 新增可测试的夜间调度入口，worker 启动轻量定时器调用；存储层提供“最近活跃用户”查询。

**Tech Stack:** Go、Kratos、PostgreSQL、Asynq

---

### Task 1: 存储层查询“最近活跃用户”（TDD）

**Files:**
- Modify: `backend/internal/storage/postgres.go`
- Test: `backend/internal/storage/postgres_baseline_test.go`

**Step 1: Write the failing test**

在 `postgres_baseline_test.go` 新增：

```go
func TestListActiveUsersSince(t *testing.T) {
    dsn := os.Getenv("STRIDEWISE_TEST_DSN")
    if dsn == "" { t.Skip("STRIDEWISE_TEST_DSN not set") }

    pool, _ := pgxpool.New(context.Background(), dsn)
    defer pool.Close()
    store := NewPostgresStore(pool)

    now := time.Now().UTC()
    // 插入一条训练记录与一条活动
    _, _ = pool.Exec(context.Background(), `INSERT INTO training_logs (log_id, user_id, source, training_type, start_time, duration_sec, distance_km, pace_str, pace_sec_per_km, rpe, discomfort, created_at, updated_at)
        VALUES ('log-1','u1','manual','easy',NOW(),1800,5,'05''30''',330,5,false,NOW(),NOW())`)
    _, _ = pool.Exec(context.Background(), `INSERT INTO activities (user_id, source, source_activity_id, name, distance_m, moving_time_sec, start_time_utc, start_time_local, timezone)
        VALUES ('u2','keep','a1','run',5000,1500,NOW(),NOW(),'UTC')`)

    users, err := store.ListActiveUsersSince(context.Background(), now.Add(-28*24*time.Hour))
    if err != nil { t.Fatalf("list active users failed: %v", err) }
    if len(users) == 0 { t.Fatalf("expected users") }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`  
Expected: FAIL（方法不存在）

**Step 3: Write minimal implementation**

- `PostgresStore.ListActiveUsersSince(ctx, since)`：
  - 查询 `training_logs` 与 `activities` 两表
  - `WHERE start_time >= since` / `start_time_local >= since`
  - `SELECT DISTINCT user_id` 并去重

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`

**Step 5: Commit**

```bash
git add backend/internal/storage/postgres.go backend/internal/storage/postgres_baseline_test.go

git commit -m "新增最近活跃用户查询"
```

---

### Task 2: 夜间基线重算入口（TDD）

**Files:**
- Create: `backend/internal/baseline/nightly.go`
- Test: `backend/internal/baseline/nightly_test.go`

**Step 1: Write the failing test**

```go
type nightlyStoreStub struct { users []string }
func (s nightlyStoreStub) ListActiveUsersSince(_ context.Context, _ time.Time) ([]string, error) { return s.users, nil }

type nightlyEnqueuerStub struct { calls int; lastUser string; lastTrigger string }
func (e *nightlyEnqueuerStub) EnqueueBaselineRecalc(_ context.Context, userID, triggerType, triggerRef string) error {
    e.calls++; e.lastUser = userID; e.lastTrigger = triggerType; return nil
}

func TestRunNightlyBaselineRecalc(t *testing.T) {
    store := nightlyStoreStub{users: []string{"u1"}}
    enq := &nightlyEnqueuerStub{}
    now := time.Date(2026, 3, 11, 2, 0, 0, 0, time.Local)
    RunNightlyBaselineRecalc(context.Background(), store, enq, func() time.Time { return now })
    if enq.calls != 1 || enq.lastUser != "u1" || enq.lastTrigger != "nightly" {
        t.Fatalf("expected nightly enqueue")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/baseline -v`  
Expected: FAIL（函数不存在）

**Step 3: Write minimal implementation**

- `RunNightlyBaselineRecalc(ctx, store, enqueuer, nowFn)`
  - `since = nowFn().Add(-28*24*time.Hour)`
  - `users, _ := store.ListActiveUsersSince(since)`
  - 遍历用户调用 `EnqueueBaselineRecalc`，`trigger_type=nightly`，`trigger_ref=nightly-YYYYMMDD`

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/baseline -v`

**Step 5: Commit**

```bash
git add backend/internal/baseline/nightly.go backend/internal/baseline/nightly_test.go

git commit -m "新增夜间基线重算入口"
```

---

### Task 3: Worker 定时触发（TDD）

**Files:**
- Modify: `backend/cmd/worker/main.go`
- Test: `backend/cmd/worker/main_test.go`

**Step 1: Write the failing test**

```go
func TestNextNightlyDelay(t *testing.T) {
    now := time.Date(2026, 3, 11, 1, 0, 0, 0, time.Local)
    d := nextNightlyDelay(now)
    if d <= 0 { t.Fatalf("expected positive delay") }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/cmd/worker -v`  
Expected: FAIL（函数不存在）

**Step 3: Write minimal implementation**

- 增加 `nextNightlyDelay(now time.Time) time.Duration`
  - 计算下一个本地 02:00 的时间差
- `main` 中启动 goroutine：
  - `for { time.Sleep(nextNightlyDelay(time.Now())); baseline.RunNightlyBaselineRecalc(...) }`

**Step 4: Run test to verify it passes**

Run: `go test ./backend/cmd/worker -v`

**Step 5: Commit**

```bash
git add backend/cmd/worker/main.go backend/cmd/worker/main_test.go

git commit -m "worker 增加夜间基线调度"
```

---

### Task 4: 全量回归与飞行记录

**Step 1: Run full tests**

Run: `go test ./backend/...`  
Expected: PASS

**Step 2: Flight Recorder**

```bash
python3 /Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py "Feature" "新增夜间基线重算调度" "夜间触发入队可能增加队列负载；已回归测试" "S2"
```

**Step 3: Commit**

```bash
git add docs/AI_CHANGELOG.md

git commit -m "记录夜间基线重算"
```
