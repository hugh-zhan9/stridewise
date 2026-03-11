# Training Recalc Orchestration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为训练记录更新/删除补齐 `training:recalc` 编排与 manual 限制，确保基线重算、建议刷新、总结/反馈回滚链路完整。

**Architecture:** 训练变更统一入队 `training:recalc`，训练处理器负责调用基线重算与建议生成；基线处理器暴露无任务状态的 Recalc 方法复用已有计算与总结回滚逻辑。接口层校验仅允许 `manual` 来源更新/删除。

**Tech Stack:** Go, Kratos, Asynq, PostgreSQL

---

### Task 1: 训练重算编排单测（TDD）

**Files:**
- Modify: `backend/internal/training/processor_test.go`

**Step 1: Write the failing test**

新增用例，验证 `training:recalc` 会调用基线重算与建议刷新，并在 `summaryErr` 时仍标记 success 且写入 errMsg。

```go
func TestProcessor_ProcessTrainingRecalc_InvokesDeps(t *testing.T) {
    store := &fakeAsyncStore{}
    baseline := &fakeBaseline{summaryErr: errors.New("summary failed")}
    rec := &fakeRecommender{}
    p := NewProcessor(store, baseline, rec)

    if err := p.ProcessTrainingRecalc(context.Background(), "job-1", "u1", "log-1", "update", 2); err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
    if !baseline.called {
        t.Fatalf("expected baseline recalc called")
    }
    if !rec.called {
        t.Fatalf("expected recommendation generate called")
    }
    if store.lastStatus != "success" {
        t.Fatalf("expected success, got %s", store.lastStatus)
    }
    if store.lastErr == "" {
        t.Fatalf("expected err msg recorded")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/training -run TestProcessor_ProcessTrainingRecalc_InvokesDeps -v`
Expected: FAIL (NewProcessor 参数不匹配或调用未发生)

**Step 3: Write minimal implementation**

- 更新 `training.Processor` 增加 `baseline` 与 `recommender` 依赖。
- 在 `ProcessTrainingRecalc` 内调用基线重算与建议刷新。
- `summaryErr` 不作为失败，仅记录 `errMsg`。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/training -run TestProcessor_ProcessTrainingRecalc_InvokesDeps -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/training/processor.go backend/internal/training/processor_test.go
git commit -m "完善训练重算编排"
```

### Task 2: 基线处理器暴露无任务状态重算

**Files:**
- Modify: `backend/internal/baseline/processor.go`
- Test: `backend/internal/baseline/processor_test.go`

**Step 1: Write the failing test**

新增测试验证 `RecalcForTrigger` 调用成功并返回 `(summaryErr, err)`。

```go
func TestProcessor_RecalcForTrigger(t *testing.T) {
    store := &fakeStore{ /* 最小可用 */ }
    p := NewProcessor(store)
    _, err := p.RecalcForTrigger(context.Background(), "u1", "training_update", "log-1")
    if err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/baseline -run TestProcessor_RecalcForTrigger -v`
Expected: FAIL (method undefined)

**Step 3: Write minimal implementation**

在 `baseline.Processor` 增加导出方法，直接调用内部 `recalc`：

```go
func (p *Processor) RecalcForTrigger(ctx context.Context, userID, triggerType, triggerRef string) (error, error) {
    return p.recalc(ctx, userID, triggerType, triggerRef)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/baseline -run TestProcessor_RecalcForTrigger -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/baseline/processor.go backend/internal/baseline/processor_test.go
git commit -m "暴露基线重算复用接口"
```

### Task 3: worker 注入训练重算依赖

**Files:**
- Modify: `backend/cmd/worker/main.go`

**Step 1: Write the failing test**

在 `backend/cmd/worker/main_test.go` 增加用例，验证训练处理器初始化时依赖不为 nil。

```go
func TestWorkerWiring_TrainingProcessor(t *testing.T) {
    // 参考现有测试结构，验证 SetTrainingProcessor 使用的是带依赖的新实例
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/worker -run TestWorkerWiring_TrainingProcessor -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- 在 worker 启动时构建 weather provider 与 recommendation processor。
- 使用 `training.NewProcessor(store, baselineProcessor, recProcessor)` 注入依赖。

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/worker -run TestWorkerWiring_TrainingProcessor -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/worker/main.go backend/cmd/worker/main_test.go
git commit -m "worker 注入训练重算依赖"
```

### Task 4: 接口限制 manual 更新/删除 + 入队 training:recalc

**Files:**
- Modify: `backend/internal/server/http.go`
- Modify: `backend/internal/server/http_training_test.go`

**Step 1: Write the failing test**

新增用例：当 `log.source != manual` 时，PUT/DELETE 返回 400。

```go
func TestUpdateTrainingLog_RejectNonManual(t *testing.T) {
    // store.GetTrainingLog 返回 source="strava"
    // 期望 400
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server -run TestUpdateTrainingLog_RejectNonManual -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- PUT/DELETE 前先读取 log，若 `source != manual` 直接 400。
- 新增/更新/删除改为入队 `training:recalc`（使用 `enqueueTrainingRecalc`）。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/server -run TestUpdateTrainingLog_RejectNonManual -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/server/http.go backend/internal/server/http_training_test.go
git commit -m "限制手动记录更新删除并改用训练重算"
```

### Task 5: 回归测试

**Files:**
- Test: `backend/internal/training`
- Test: `backend/internal/baseline`
- Test: `backend/internal/server`

**Step 1: Run tests**

Run: `go test ./internal/training ./internal/baseline ./internal/server -v`
Expected: PASS

**Step 2: Commit (if any fixes required)**

```bash
git add -A
git commit -m "修复训练重算链路回归测试"
```
