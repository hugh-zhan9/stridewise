# 恢复状态规则 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 基于 ACWR/Monotony 判定恢复状态，作为规则输入触发安全降级与替代方案。

**Architecture:** 在推荐生成时计算恢复状态并写入 input/output；规则层新增 recovery 判定；AI 输出保持可选字段，规则兜底确保安全。

**Tech Stack:** Go, Kratos, Postgres

---

### Task 1: 增加恢复状态计算函数

**Files:**
- Create: `backend/internal/recommendation/recovery.go`
- Test: `backend/internal/recommendation/recovery_test.go`

**Step 1: Write the failing test**

```go
func TestRecoveryStatus_ByACWR(t *testing.T) {
	got := CalcRecoveryStatus(1.6, 1.0)
	if got != "red" {
		t.Fatalf("expected red")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`
Expected: FAIL（函数不存在）

**Step 3: Write minimal implementation**

实现：
- `CalcRecoveryStatus(acwr, monotony float64) string`
- 阈值：ACWR >1.5 red；>1.3 yellow；Monotony >=2.2 red；>=2.0 yellow

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/recommendation/recovery.go backend/internal/recommendation/recovery_test.go

git commit -m "新增恢复状态计算"
```

---

### Task 2: 推荐输入新增恢复状态字段

**Files:**
- Modify: `backend/internal/ai/recommender.go`
- Modify: `backend/internal/recommendation/processor.go`
- Test: `backend/internal/recommendation/processor_test.go`

**Step 1: Write the failing test**

在 `processor_test.go` 中断言 `input_json` 含 `recovery_status`。

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- `RecommendationInput` 增加 `RecoveryStatus` 字段
- `Processor.Generate` 计算 recovery 并写入 input

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/ai/recommender.go backend/internal/recommendation/processor.go backend/internal/recommendation/processor_test.go

git commit -m "推荐输入包含恢复状态"
```

---

### Task 3: 规则层引入恢复状态

**Files:**
- Modify: `backend/internal/recommendation/rules.go`
- Modify: `backend/internal/recommendation/rules_test.go`

**Step 1: Write the failing test**

```go
func TestApplyRules_RecoveryRedOverrides(t *testing.T) {
	out := RecommendationOutput{ShouldRun: true}
	res := ApplyRules(RuleInput{RecoveryStatus: "red"}, out)
	if res.Output.ShouldRun {
		t.Fatalf("expected override")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- `RuleInput` 增加 `RecoveryStatus` 字段
- 规则优先级：安全 > 恢复 > 天气 > 高负荷

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/recommendation/rules.go backend/internal/recommendation/rules_test.go

git commit -m "恢复不足触发降级"
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

补充 `recovery_status` 字段说明。

**Step 4: Run test to verify it passes**

无

**Step 5: Commit**

```bash
git add docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md

git commit -m "文档补充恢复状态字段"
```

