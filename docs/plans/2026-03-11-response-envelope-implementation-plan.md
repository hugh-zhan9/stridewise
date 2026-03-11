# 异常兜底响应标准化 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 统一内部 API 响应包裹结构，标准化错误与兜底元信息输出。

**Architecture:** 在 HTTP 层新增统一响应封装与错误输出，替换所有 `writeJSON/http.Error`；在推荐/天气等兜底场景填充 `fallback_reason/confidence`；同步 OpenAPI 与测试。

**Tech Stack:** Go、Kratos HTTP、PostgreSQL

---

### Task 1: 统一响应封装与错误输出（HTTP 层）

**Files:**
- Modify: `backend/internal/server/http.go`
- Test: `backend/internal/server/http_baseline_test.go`

**Step 1: Write the failing test**

在 `http_baseline_test.go` 新增最小验证：

```go
func TestResponseEnvelopeSuccess(t *testing.T) {
    srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, &baselineStoreStub{}, nil, nil)
    req := httptest.NewRequest(http.MethodGet, "/internal/v1/training/summaries?user_id=u1", nil)
    req.Header.Set("X-Internal-Token", "token")
    rec := httptest.NewRecorder()
    srv.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }
    body := rec.Body.String()
    if !strings.Contains(body, "\"data\"") || !strings.Contains(body, "\"meta\"") {
        t.Fatalf("expected envelope, got %s", body)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/server -v`
Expected: FAIL（现有输出无包裹）

**Step 3: Write minimal implementation**

- 在 `http.go` 新增响应结构体：
  - `responseEnvelope` / `responseMeta` / `responseError`
- 新增函数：
  - `writeSuccess(w, status, data, meta)`
  - `writeError(w, status, code, message, meta)`
  - `buildMeta(r, fallbackReason, confidence)`
  - `errorCodeForStatus(status)`
- 替换所有 `writeJSON` 调用为 `writeSuccess`。
- 替换所有 `http.Error` 为 `writeError`（保持原状态码）。
- 保留原 `writeJSON`，但内部改为 `writeSuccess`（可用于兼容代码）。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/server -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_baseline_test.go

git commit -m "统一内部API响应包裹"
```

---

### Task 2: 兜底元信息与错误码映射

**Files:**
- Modify: `backend/internal/server/http.go`
- Modify: `backend/internal/server/http_recommendation_ability_test.go`

**Step 1: Write the failing test**

在推荐接口测试中断言兜底信息：

```go
func TestGenerateRecommendationFallbackMeta(t *testing.T) {
    store := &fakeStore{profile: storage.UserProfile{UserID: "u1", LocationLat: 1, LocationLng: 2, Country: "CN", Province: "SH", City: "SH", AbilityLevel: "beginner"}}
    p := recommendation.NewProcessor(store, fakeWeather{}, fakeAI{})
    srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, store, nil, p)

    req := httptest.NewRequest(http.MethodPost, "/internal/v1/recommendations/generate", strings.NewReader(`{"user_id":"u1"}`))
    req.Header.Set("X-Internal-Token", "token")
    rec := httptest.NewRecorder()
    srv.ServeHTTP(rec, req)

    body := rec.Body.String()
    if !strings.Contains(body, "fallback_reason") {
        t.Fatalf("expected fallback_reason in meta")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/server -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- 在推荐接口返回时：
  - `rec.IsFallback == true` → `fallback_reason = "ai_unavailable"`，`confidence = 0.4`。
  - `rec.OverrideJSON` 非空 → `fallback_reason = "safety_override"`，`confidence = 0.6`。
- `ErrAbilityLevelNotReady` → `error.code = dependency_unavailable`，`fallback_reason = "ability_level_not_ready"`。
- 其他错误保持 `internal_error`，`fallback_reason` 为空。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/server -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_recommendation_ability_test.go

git commit -m "补充兜底元信息输出"
```

---

### Task 3: 文档同步

**Files:**
- Modify: `docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`

**Step 1: Update OpenAPI**

- 新增统一响应包裹 `Envelope`/`EnvelopeError`/`EnvelopeMeta` schema。
- 全部 `/internal/**` 响应 `schema` 用 `Envelope` 包裹。
- `fallback_reason` 与 `confidence` 写入 `EnvelopeMeta`。

**Step 2: Commit**

```bash
git add docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md

git commit -m "文档补充响应包裹"
```

---

### Task 4: 全量回归与飞行记录

**Step 1: Run full tests**

Run: `go test ./backend/...`
Expected: PASS

**Step 2: Flight Recorder**

```bash
python3 /Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py "Feature" "统一内部API响应包裹并补充兜底元信息" "风险在于调用方响应解析与测试断言变更，已回归验证" "S2"
```

**Step 3: Commit**

```bash
git add docs/AI_CHANGELOG.md

git commit -m "记录响应包裹变更"
```
