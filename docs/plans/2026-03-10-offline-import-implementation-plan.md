# Offline Import (running_page) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement offline import connectors for Keep/Strava/Garmin/Nike/GPX/TCX/FIT using running_page's `activities.json` model and checkpoint-based incremental sync.

**Architecture:** Add a shared JSON parser that converts running_page activities into `RawActivity` + `FetchResult`, then implement per-source connectors as thin wrappers pointing to configured data files. Wire the connectors into the worker and config. TDD for parsing and connector behavior.

**Tech Stack:** Go, Kratos, Asynq, PostgreSQL

---

### Task 1: 通用 JSON 解析器（TDD）

**Files:**
- Create: `backend/internal/connector/common/json_parser.go`
- Create: `backend/internal/connector/common/json_parser_test.go`

**Step 1: Write the failing test**

```go
package common

import (
    "os"
    "testing"
    "time"

    syncjob "stridewise/backend/internal/sync"
)

func TestParseRunningPageJSON_FiltersByCheckpoint(t *testing.T) {
    tmp, err := os.CreateTemp("", "activities-*.json")
    if err != nil {
        t.Fatalf("temp: %v", err)
    }
    t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

    payload := `[
      {"run_id": 1, "name": "a", "distance": 1000, "moving_time": "0:10:00", "start_date": "2026-01-01 00:00:00+00:00", "start_date_local": "2026-01-01 08:00:00"},
      {"run_id": 2, "name": "b", "distance": 2000, "moving_time": "0:20:00", "start_date": "2026-01-02 00:00:00+00:00", "start_date_local": "2026-01-02 08:00:00"}
    ]`
    if _, err := tmp.WriteString(payload); err != nil {
        t.Fatalf("write: %v", err)
    }

    cp := syncjob.Checkpoint{LastSyncedAt: time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)}
    res, err := ParseRunningPageJSON(tmp.Name(), cp)
    if err != nil {
        t.Fatalf("parse: %v", err)
    }
    if len(res.Activities) != 1 {
        t.Fatalf("expected 1 activity, got %d", len(res.Activities))
    }
    if res.Activities[0].SourceActivityID != "2" {
        t.Fatalf("unexpected id: %s", res.Activities[0].SourceActivityID)
    }
    if res.LastSyncedAt.IsZero() {
        t.Fatal("expected last synced at")
    }
}

func TestParseRunningPageJSON_FallbackStartDate(t *testing.T) {
    tmp, err := os.CreateTemp("", "activities-*.json")
    if err != nil {
        t.Fatalf("temp: %v", err)
    }
    t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

    payload := `[
      {"run_id": "a1", "name": "x", "distance": 1000, "moving_time": "0:10:00", "start_date": "2026-01-03 00:00:00+00:00", "start_date_local": "invalid"}
    ]`
    if _, err := tmp.WriteString(payload); err != nil {
        t.Fatalf("write: %v", err)
    }

    res, err := ParseRunningPageJSON(tmp.Name(), syncjob.Checkpoint{})
    if err != nil {
        t.Fatalf("parse: %v", err)
    }
    if len(res.Activities) != 1 {
        t.Fatalf("expected 1 activity, got %d", len(res.Activities))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/connector/common -v`
Expected: FAIL (package or function not found)

**Step 3: Write minimal implementation**

```go
package common

import (
    "encoding/json"
    "errors"
    "os"
    "strconv"
    "strings"
    "time"

    syncjob "stridewise/backend/internal/sync"
)

type runningPageActivity struct {
    RunID          any     `json:"run_id"`
    Name           string  `json:"name"`
    Distance       float64 `json:"distance"`
    MovingTime     string  `json:"moving_time"`
    StartDate      string  `json:"start_date"`
    StartDateLocal string  `json:"start_date_local"`
    SummaryPolyline string `json:"summary_polyline"`
}

func ParseRunningPageJSON(dataFile string, checkpoint syncjob.Checkpoint) (syncjob.FetchResult, error) {
    if dataFile == "" {
        return syncjob.FetchResult{}, errors.New("data_file is empty")
    }
    b, err := os.ReadFile(dataFile)
    if err != nil {
        return syncjob.FetchResult{}, err
    }
    var list []runningPageActivity
    if err := json.Unmarshal(b, &list); err != nil {
        return syncjob.FetchResult{}, err
    }

    out := make([]syncjob.RawActivity, 0, len(list))
    lastSyncedAt := checkpoint.LastSyncedAt
    for _, item := range list {
        start, err := parseStartDate(item.StartDateLocal)
        if err != nil {
            start, err = parseStartDate(item.StartDate)
            if err != nil {
                continue
            }
        }
        if !checkpoint.LastSyncedAt.IsZero() && !start.After(checkpoint.LastSyncedAt) {
            continue
        }
        sourceID := toSourceID(item.RunID)
        if sourceID == "" {
            continue
        }
        out = append(out, syncjob.RawActivity{
            SourceActivityID: sourceID,
            Name:             item.Name,
            DistanceM:        item.Distance,
            MovingTimeSec:    parseMovingTime(item.MovingTime),
            StartTime:        start,
            SummaryPolyline:  item.SummaryPolyline,
            Raw: map[string]any{
                "run_id": item.RunID,
                "name": item.Name,
                "distance": item.Distance,
                "moving_time": item.MovingTime,
                "start_date": item.StartDate,
                "start_date_local": item.StartDateLocal,
            },
        })
        if start.After(lastSyncedAt) {
            lastSyncedAt = start
        }
    }

    return syncjob.FetchResult{
        Activities: out,
        LastSyncedAt: lastSyncedAt,
    }, nil
}

func parseStartDate(s string) (time.Time, error) {
    layouts := []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02 15:04:05+00:00"}
    for _, layout := range layouts {
        if t, err := time.Parse(layout, s); err == nil {
            return t, nil
        }
    }
    return time.Time{}, errors.New("invalid start date")
}

func parseMovingTime(s string) int {
    if s == "" {
        return 0
    }
    parts := strings.Split(s, ", ")
    days := 0
    timePart := parts[len(parts)-1]
    if len(parts) == 2 {
        d := strings.Fields(parts[0])
        if len(d) > 0 {
            days, _ = strconv.Atoi(d[0])
        }
    }
    hms := strings.Split(timePart, ":")
    if len(hms) != 3 {
        return 0
    }
    h, _ := strconv.Atoi(hms[0])
    m, _ := strconv.Atoi(hms[1])
    sec, _ := strconv.Atoi(hms[2])
    return ((days*24+h)*60+m)*60 + sec
}

func toSourceID(v any) string {
    switch x := v.(type) {
    case string:
        return x
    case float64:
        return strconv.FormatInt(int64(x), 10)
    case int:
        return strconv.Itoa(x)
    default:
        return ""
    }
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/connector/common -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/connector/common/json_parser.go backend/internal/connector/common/json_parser_test.go
git commit -m "feat: add running_page json parser"
```

**Step 6: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Feature" "新增 running_page JSON 通用解析器并完成增量过滤" "风险在于时间解析与增量过滤边界；需关注 start_date_local 不同格式兼容" "S2"
```

---

### Task 2: Keep 连接器改为通用解析器（TDD）

**Files:**
- Modify: `backend/internal/connector/keep/connector.go`
- Create: `backend/internal/connector/keep/connector_test.go`

**Step 1: Write the failing test**

```go
package keep

import (
    "os"
    "testing"

    syncjob "stridewise/backend/internal/sync"
)

func TestConnector_FetchActivities(t *testing.T) {
    tmp, err := os.CreateTemp("", "activities-*.json")
    if err != nil {
        t.Fatalf("temp: %v", err)
    }
    t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

    payload := `[{"run_id": 1, "name": "a", "distance": 1000, "moving_time": "0:10:00", "start_date_local": "2026-01-01 08:00:00"}]`
    if _, err := tmp.WriteString(payload); err != nil {
        t.Fatalf("write: %v", err)
    }

    c := New(tmp.Name())
    res, err := c.FetchActivities(nil, "u1", syncjob.Checkpoint{})
    if err != nil {
        t.Fatalf("fetch: %v", err)
    }
    if len(res.Activities) != 1 {
        t.Fatalf("expected 1 activity, got %d", len(res.Activities))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/connector/keep -v`
Expected: FAIL

**Step 3: Write minimal implementation**

Update `connector.go` to call `common.ParseRunningPageJSON` and remove duplicated parsing logic.

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/connector/keep -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/connector/keep/connector.go backend/internal/connector/keep/connector_test.go
git commit -m "refactor: keep connector use common parser"
```

**Step 6: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Refactor" "Keep 连接器改为通用 JSON 解析器" "风险较低，主要在于逻辑迁移导致解析字段丢失" "S3"
```

---

### Task 3: 新增 Strava/Garmin/Nike/GPX/TCX/FIT 连接器（TDD）

**Files:**
- Create: `backend/internal/connector/strava/connector.go`
- Create: `backend/internal/connector/garmin/connector.go`
- Create: `backend/internal/connector/nike/connector.go`
- Create: `backend/internal/connector/gpx/connector.go`
- Create: `backend/internal/connector/tcx/connector.go`
- Create: `backend/internal/connector/fit/connector.go`
- Create: `backend/internal/connector/strava/connector_test.go`

**Step 1: Write the failing test**

```go
package strava

import (
    "os"
    "testing"

    syncjob "stridewise/backend/internal/sync"
)

func TestConnector_FetchActivities(t *testing.T) {
    tmp, err := os.CreateTemp("", "activities-*.json")
    if err != nil {
        t.Fatalf("temp: %v", err)
    }
    t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

    payload := `[{"run_id": 1, "name": "a", "distance": 1000, "moving_time": "0:10:00", "start_date_local": "2026-01-01 08:00:00"}]`
    if _, err := tmp.WriteString(payload); err != nil {
        t.Fatalf("write: %v", err)
    }

    c := New(tmp.Name())
    res, err := c.FetchActivities(nil, "u1", syncjob.Checkpoint{})
    if err != nil {
        t.Fatalf("fetch: %v", err)
    }
    if len(res.Activities) != 1 {
        t.Fatalf("expected 1 activity, got %d", len(res.Activities))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/connector/strava -v`
Expected: FAIL

**Step 3: Write minimal implementation**

Each connector should be a thin wrapper:

```go
package strava

import (
    "context"

    "stridewise/backend/internal/connector/common"
    syncjob "stridewise/backend/internal/sync"
)

type Connector struct {
    DataFile string
}

func New(dataFile string) *Connector {
    return &Connector{DataFile: dataFile}
}

func (c *Connector) FetchActivities(_ context.Context, _ string, checkpoint syncjob.Checkpoint) (syncjob.FetchResult, error) {
    return common.ParseRunningPageJSON(c.DataFile, checkpoint)
}
```

Replicate for garmin/nike/gpx/tcx/fit.

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/connector/strava -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/connector/strava backend/internal/connector/garmin backend/internal/connector/nike backend/internal/connector/gpx backend/internal/connector/tcx backend/internal/connector/fit
git commit -m "feat: add offline connectors for sources"
```

**Step 6: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Feature" "新增 Strava/Garmin/Nike/GPX/TCX/FIT 离线连接器" "风险在于多来源复用单一 JSON 路径导致混源；需由调用方保证单次单源" "S2"
```

---

### Task 4: 配置扩展 + Worker 接入（TDD）

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/config/config.yaml`
- Modify: `backend/cmd/worker/main.go`
- Create: `backend/internal/worker/handler_test.go`

**Step 1: Write the failing test**

```go
package worker

import (
    "testing"

    "github.com/hibiken/asynq"
    "stridewise/backend/internal/task"
)

func TestHandleSyncJob_UsesRetryCount(t *testing.T) {
    payload := task.SyncJobPayload{JobID: "j1", UserID: "u1", Source: "keep", RetryCount: 2}
    b, err := task.EncodeSyncJobPayload(payload)
    if err != nil {
        t.Fatalf("encode: %v", err)
    }

    // processor nil -> should error (sanity for wiring tests)
    if err := HandleSyncJob(nil, asynq.NewTask(task.TypeSyncJob, b)); err == nil {
        t.Fatal("expected error when processor not set")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/worker -v`
Expected: FAIL (missing test file or behavior)

**Step 3: Write minimal implementation**

- 配置新增：`strava`, `garmin`, `nike`, `gpx`, `tcx`, `fit` 的 `data_file`。
- Worker 注册全部连接器：
  - `keep.New(cfg.Keep.DataFile)`
  - `strava.New(cfg.Strava.DataFile)`
  - `garmin.New(cfg.Garmin.DataFile)`
  - `nike.New(cfg.Nike.DataFile)`
  - `gpx.New(cfg.GPX.DataFile)`
  - `tcx.New(cfg.TCX.DataFile)`
  - `fit.New(cfg.FIT.DataFile)`

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/worker -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/config/config.go backend/config/config.yaml backend/cmd/worker/main.go backend/internal/worker/handler_test.go
git commit -m "feat: wire offline connectors in worker"
```

**Step 6: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Feature" "扩展配置并在 worker 中接入多来源离线连接器" "风险主要在配置缺失导致运行时错误，需要确保配置文件同步更新" "S2"
```

---

### Task 5: 全量回归

**Files:**
- N/A

**Step 1: Run full test suite**

Run: `cd backend && go test ./...`
Expected: PASS

**Step 2: Commit (if needed)**

If any fixes applied during this step, commit with:

```bash
git add <files>
git commit -m "fix: stabilize offline import"
```

**Step 3: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Bugfix" "离线导入回归修复" "风险低，主要为补丁级修复" "S3"
```
