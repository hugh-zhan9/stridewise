# 能力层级自动判定 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 能力层级完全由 AI 自动判定；无历史能力层级时阻断建议并异步重试；支持问卷触发与训练/同步触发。

**Architecture:** 新增 ability_level 异步任务类型与处理器，任务成功写回 user_profiles；建议生成前检查 ability_level，若缺失则阻断并创建任务。

**Tech Stack:** Go, Kratos, Postgres, Asynq, OpenAI

---

### Task 1: UserProfile 能力层级字段落库

**Files:**
- Modify: `backend/migrations/003_user_weather.sql`
- Modify: `backend/internal/storage/postgres.go`
- Modify: `backend/internal/storage/postgres_user_weather_test.go`

**Step 1: Write the failing test**

在 `postgres_user_weather_test.go` 追加能力层级字段断言：

```go
assertStringField(t, got, "AbilityLevel", "beginner")
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`
Expected: FAIL（字段不存在或未写入）

**Step 3: Write minimal implementation**

- 迁移新增或替换 `ability_level`、`ability_level_updated_at`、`ability_level_reason`
- `UserProfile` 结构体新增字段
- `UpsertUserProfile` / `GetUserProfile` SQL 增加字段

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/migrations/003_user_weather.sql backend/internal/storage/postgres.go backend/internal/storage/postgres_user_weather_test.go

git commit -m "用户档案新增能力层级字段"
```

---

### Task 2: Profile 接口移除手动能力层级输入

**Files:**
- Modify: `backend/internal/server/http.go`
- Modify: `backend/internal/server/http_user_weather_test.go`

**Step 1: Write the failing test**

新增测试：请求体包含 `fitness_level` 应返回 400；`ability_level` 不允许写入。

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/server -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- `userProfileRequest` 移除 `fitness_level` 字段
- 校验逻辑：若检测到 `fitness_level`（旧字段）则返回 400
- Profile 响应中返回 `ability_level` 与 `ability_level_updated_at`

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/server -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_user_weather_test.go

git commit -m "用户档案接口移除手动能力层级"
```

---

### Task 3: Ability Level 异步任务与入队

**Files:**
- Modify: `backend/internal/task/task.go`
- Modify: `backend/internal/task/task_test.go`
- Create: `backend/internal/asyncjob/ability_level_enqueuer.go`
- Modify: `backend/internal/storage/postgres.go`
- Modify: `backend/internal/storage/postgres_training_test.go`

**Step 1: Write the failing test**

- task：新增 payload 编解码测试
- storage：新增查询用户 active 异步任务测试（queued/running）

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/task -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- 新任务类型 `ability_level:calc`
- payload 结构：`job_id/user_id/trigger_type/trigger_ref`
- 新增 `FindActiveAsyncJob`（按 user_id + job_type + status）
- AbilityLevelEnqueuer：若存在 active 任务则复用，否者创建并入队

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/task -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/task/task.go backend/internal/task/task_test.go backend/internal/asyncjob/ability_level_enqueuer.go backend/internal/storage/postgres.go backend/internal/storage/postgres_training_test.go

git commit -m "新增能力层级异步任务类型"
```

---

### Task 4: Ability Level Processor 与 Worker 处理

**Files:**
- Create: `backend/internal/ability/processor.go`
- Create: `backend/internal/ability/processor_test.go`
- Create: `backend/internal/ai/ability.go`
- Create: `backend/internal/ai/openai_ability.go`
- Modify: `backend/internal/worker/handler.go`
- Modify: `backend/cmd/worker/main.go`

**Step 1: Write the failing test**

在 `processor_test.go` 中添加用例：
- AI 成功返回 → user_profiles 写入 ability_level
- AI 失败 → 不覆盖已有 ability_level

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/ability -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- Ability Processor：
  - 拉取 28 天训练日志/活动，构建 AI 输入
  - 调用 AI 能力层级接口
  - 更新 `ability_level`、`ability_level_updated_at`、`ability_level_reason`
  - 失败时仅更新 async_jobs 状态，不覆盖现值
- Worker 注册新 handler：`HandleAbilityLevelCalc`
- Worker main 绑定 handler 与 processor

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/ability -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/ability backend/internal/ai/ability.go backend/internal/ai/openai_ability.go backend/internal/worker/handler.go backend/cmd/worker/main.go

git commit -m "新增能力层级任务处理器"
```

---

### Task 5: 建议生成阻断与能力层级检查

**Files:**
- Modify: `backend/internal/server/http.go`
- Modify: `backend/internal/recommendation/processor.go`
- Modify: `backend/internal/recommendation/processor_test.go`

**Step 1: Write the failing test**

- recommendation 生成时 `ability_level` 为空 → 返回错误并触发 ability_level 任务

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/server -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- `recommendations/generate` 前检查 user_profiles.ability_level
- 为空时 enqueue `ability_level:calc` 并返回 `ability_level_not_ready`
- Processor 内部禁止空能力层级写入 AI 输入

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/server -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/recommendation/processor.go backend/internal/recommendation/processor_test.go

git commit -m "建议生成加入能力层级阻断"
```

---

### Task 6: 文档同步

**Files:**
- Modify: `request.md`
- Modify: `docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`
- Modify: `docs/plans/2026-03-10-user-weather-design.md`

**Step 1: Write the failing test**

无

**Step 2: Run test to verify it fails**

无

**Step 3: Write minimal implementation**

- 删除“能力层级支持手动调整”描述（使用删除线标注）
- OpenAPI/Profile schema 移除 `fitness_level` 输入
- 同步文档版本号与变更记录

**Step 4: Run test to verify it passes**

无

**Step 5: Commit**

```bash
git add request.md docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md docs/plans/2026-03-10-user-weather-design.md

git commit -m "文档同步能力层级自动判定"
```

