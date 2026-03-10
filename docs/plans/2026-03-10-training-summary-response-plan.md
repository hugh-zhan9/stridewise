# 训练总结接口响应字段统一 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** `GET /internal/v1/training/summaries` 返回字段统一为 `snake_case`，与 OpenAPI 一致。

**Architecture:** 在 HTTP 层新增响应 DTO，显式 JSON tag 映射字段名，将 storage 层结构体映射后输出。

**Tech Stack:** Go、Kratos HTTP

---

### Task 1: 训练总结响应 DTO 与映射

**Files:**
- Modify: `backend/internal/server/http.go`
- Modify: `backend/internal/server/http_baseline_test.go`

**Step 1: Write the failing test**

Add test in `http_baseline_test.go`:

```go
func TestListTrainingSummariesResponseSnakeCase(t *testing.T) {
    srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, &baselineStoreStubWithSummary{}, nil)

    req := httptest.NewRequest(http.MethodGet, "/internal/v1/training/summaries?user_id=u1", nil)
    req.Header.Set("X-Internal-Token", "token")
    rec := httptest.NewRecorder()
    srv.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }
    body := rec.Body.String()
    if !strings.Contains(body, "summary_id") || !strings.Contains(body, "source_type") || !strings.Contains(body, "source_id") {
        t.Fatalf("expected snake_case fields, got %s", body)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/server -v`
Expected: FAIL (still returns camelCase)

**Step 3: Write minimal implementation**

- In `http.go`, add `trainingSummaryResponse` DTO with json tags.
- Map `[]storage.TrainingSummary` to `[]trainingSummaryResponse` before `writeJSON`.
- Time fields use `time.RFC3339`.

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/server -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_baseline_test.go
git commit -m "训练总结响应字段统一"
```

---

### Task 2: 全量回归与变更记录

**Files:**
- Modify: `docs/AI_CHANGELOG.md`

**Step 1: Run full tests**

Run: `go test ./backend/...`
Expected: PASS

**Step 2: Flight Recorder**

Run:
```bash
python3 /Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py "Feature" "训练总结接口响应字段统一为snake_case" "仅影响训练总结列表输出字段名，若前端依赖旧字段会失败" "S2"
```

**Step 3: Commit**

```bash
git add docs/AI_CHANGELOG.md
git commit -m "记录训练总结响应字段变更"
```
