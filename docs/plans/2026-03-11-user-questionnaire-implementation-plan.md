# 新用户问卷与保守模板 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 新用户问卷字段入库，并在训练数据不足时输出保守建议模板。

**Architecture:** 在 user_profiles 扩展问卷字段；推荐流程加入“数据不足 → 保守模板”分支；使用现有 API。 

**Tech Stack:** Go, Kratos, Postgres

---

### Task 1: 扩展 user_profiles 字段与存储

**Files:**
- Modify: `backend/migrations/003_user_weather.sql`
- Modify: `backend/internal/storage/postgres.go`
- Modify: `backend/internal/storage/postgres_user_weather_test.go`

**Step 1: Write the failing test**

在 `postgres_user_weather_test.go` 中新增字段断言：

```go
if got.RunningYears != "1-3" { t.Fatalf("expected running_years") }
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`
Expected: FAIL（字段不存在）

**Step 3: Write minimal implementation**

- 迁移新增列：`running_years`、`weekly_sessions`、`weekly_distance_km`、`longest_run_km`、`recent_discomfort`
- `UserProfile` 结构体补齐字段
- `UpsertUserProfile` / `GetUserProfile` SQL 增加字段

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/migrations/003_user_weather.sql backend/internal/storage/postgres.go backend/internal/storage/postgres_user_weather_test.go

git commit -m "用户档案新增问卷字段"
```

---

### Task 2: API 写入问卷字段

**Files:**
- Modify: `backend/internal/server/http.go`
- Modify: `backend/internal/server/http_user_weather_test.go`

**Step 1: Write the failing test**

在 `http_user_weather_test.go` 中新增断言：请求写入问卷字段后 `UserProfile` 结构包含对应值。

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/server -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- `user_profile` 请求体新增字段
- 校验枚举范围

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/server -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_user_weather_test.go

git commit -m "用户档案接口支持问卷字段"
```

---

### Task 3: 保守模板输出

**Files:**
- Modify: `backend/internal/recommendation/processor.go`
- Modify: `backend/internal/recommendation/processor_test.go`

**Step 1: Write the failing test**

增加测试：
- baseline 状态为 `insufficient_data`
- 输入问卷字段
- 输出 `workout_type` 为 `easy_run` 且 `explanation` 包含“保守模板”

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- 在 `Generate` 中检测 `baseline.Status == insufficient_data` 或 `DataSessions7d < 3`
- 走 `conservativeOutput(profile)`
- 如果 `recent_discomfort=yes` → `should_run=false` + `risk_level=red`

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/recommendation/processor.go backend/internal/recommendation/processor_test.go

git commit -m "数据不足时输出保守模板"
```

---

### Task 4: 文档同步

**Files:**
- Modify: `docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`

**Step 1: Write the failing test**

无

**Step 2: Run test to verify it fails**

无

**Step 3: Write minimal implementation**

补充问卷字段 schema。

**Step 4: Run test to verify it passes**

无

**Step 5: Commit**

```bash
git add docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md

git commit -m "文档补充问卷字段"
```

