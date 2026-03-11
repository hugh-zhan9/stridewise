# Rolling Trend Analysis Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现滚动 7/30 天训练趋势分析 API，输出基础/扩展/负荷指标与按天序列。

**Architecture:** 新增 `internal/trend` 聚合器（基于现有训练记录/活动/总结/基线历史聚合），由 HTTP 层提供 `/internal/v1/trends` 查询接口，响应使用 Envelope 包裹。

**Tech Stack:** Go, Kratos HTTP, PostgreSQL

---

### Task 1: 趋势聚合器单测与实现（TDD）

**Files:**
- Create: `backend/internal/trend/processor.go`
- Create: `backend/internal/trend/types.go`
- Create: `backend/internal/trend/processor_test.go`

**Step 1: Write the failing test**

```go
func TestAggregateRollingTrend(t *testing.T) {
    store := &fakeTrendStore{
        logs: []storage.TrainingLog{{UserID: "u1", StartTime: time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC), DurationSec: 1800, DistanceKM: 5, PaceSecPerKM: 360, RPE: 6}},
        acts: []storage.Activity{{UserID: "u1", StartTimeLocal: time.Date(2026, 3, 9, 8, 0, 0, 0, time.UTC), MovingTimeSec: 1500, DistanceM: 4000}},
        summaries: []storage.TrainingSummary{{UserID: "u1", SourceType: "log", SourceID: "log-1", CompletionRate: "适中", IntensityMatch: "偏高", RecoveryAdvice: "补水"}},
        baselines: []storage.BaselineHistory{{UserID: "u1", ComputedAt: time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC), ACWRSRPE: 1.2, ACWRDistance: 1.1, Monotony: 1.8, Strain: 120}},
    }
    p := NewProcessor(store)

    out, err := p.Aggregate(context.Background(), "u1", "7d", time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC))
    if err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
    if out.Summary.Sessions != 2 {
        t.Fatalf("expected sessions 2, got %d", out.Summary.Sessions)
    }
    if out.Summary.AvgRPE <= 0 {
        t.Fatalf("expected avg rpe")
    }
    if out.Summary.CompletionRateDist["mid"] != 1 {
        t.Fatalf("expected completion_rate mid count")
    }
    if out.Summary.RecoveryAdviceTags["补水"] != 1 {
        t.Fatalf("expected recovery tag count")
    }
    if out.Summary.ACWRSRPE == nil {
        t.Fatalf("expected baseline acwr")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/trend -run TestAggregateRollingTrend -v`
Expected: FAIL (package not found)

**Step 3: Write minimal implementation**

- 定义 `TrendStore` 接口：
  - `ListTrainingLogs(ctx,userID,from,to)`
  - `ListActivities(ctx,userID,from,to)`
  - `ListTrainingSummaries(ctx,userID,from,to)`
  - `ListBaselineHistory(ctx,userID,from,to)`
- 输出结构体：`TrendSummary`、`TrendPoint`、`TrendResult`。
- 聚合逻辑：
  - sessions = logs + activities
  - avg pace = 距离加权平均
  - avg rpe = 仅 logs rpe>0
  - summaries 分布：关键词映射 low/mid/high/unknown
  - baseline 取 `as_of` 最近一条
- 关键词映射：
  - low: "低"/"不足"/"偏低"; mid: "适中"/"匹配"/"正常"; high: "高"/"过强"/"偏高"

**Step 4: Run test to verify it passes**

Run: `go test ./internal/trend -run TestAggregateRollingTrend -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/trend/processor.go backend/internal/trend/types.go backend/internal/trend/processor_test.go
git commit -m "新增滚动趋势聚合器"
```

### Task 2: HTTP 接口与参数校验（TDD）

**Files:**
- Modify: `backend/internal/server/http.go`
- Modify: `backend/internal/server/http_trend_test.go` (new)

**Step 1: Write the failing test**

```go
func TestGetTrends_RejectInvalidWindow(t *testing.T) {
    trend := &trendServiceStub{}
    srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, trend)

    req := httptest.NewRequest(http.MethodGet, "/internal/v1/trends?user_id=u1&window=14d", nil)
    req.Header.Set("X-Internal-Token", "token")
    rec := httptest.NewRecorder()
    srv.ServeHTTP(rec, req)
    if rec.Code != http.StatusBadRequest {
        t.Fatalf("expected 400")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server -run TestGetTrends_RejectInvalidWindow -v`
Expected: FAIL (handler not found)

**Step 3: Write minimal implementation**

- 增加 `TrendService` 接口：`GetRollingTrend(ctx,userID,window,asOf)`。
- 增加 handler `/internal/v1/trends`：
  - `user_id` 必填
  - `window` 仅允许 `7d`/`30d`
  - `as_of` 可为空（默认 now），支持 `YYYY-MM-DD` 或 `YYYY-MM-DD HH:MM:SS`
- 输出 `TrendResult` 走 Envelope。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/server -run TestGetTrends_RejectInvalidWindow -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_trend_test.go
git commit -m "新增趋势分析接口"
```

### Task 3: 更新 OpenAPI/Schema 文档

**Files:**
- Modify: `docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`

**Step 1: Update doc**

- 版本号 +1（Minor）。
- 新增 `/internal/v1/trends` 说明与响应结构。

**Step 2: Commit**

```bash
git add docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md
git commit -m "更新趋势分析接口文档"
```

### Task 4: 回归测试

**Files:**
- Test: `backend/internal/trend`
- Test: `backend/internal/server`

**Step 1: Run tests**

Run: `go test ./internal/trend ./internal/server -v`
Expected: PASS

**Step 2: Commit (if any fixes required)**

```bash
git add -A
git commit -m "修复趋势分析回归测试"
```
