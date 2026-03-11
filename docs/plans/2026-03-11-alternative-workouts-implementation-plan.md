# 安全降级替代运动建议 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 触发安全降级时自动输出替代运动建议，保证推荐结果始终可执行。

**Architecture:** 在推荐输出结构中新增替代方案字段，AI 可选输出但不强制；规则兜底在安全降级时补齐默认替代建议。

**Tech Stack:** Go, Kratos, Postgres

---

### Task 1: 扩展推荐输出结构（alternative_workouts）

**Files:**
- Modify: `backend/internal/recommendation/types.go`
- Modify: `backend/internal/ai/recommender.go`
- Modify: `backend/internal/recommendation/processor.go`
- Test: `backend/internal/recommendation/processor_test.go`

**Step 1: Write the failing test**

在 `processor_test.go` 增加断言：当规则触发安全降级时，输出包含 `alternative_workouts`。

```go
if len(out.AlternativeWorkouts) == 0 {
	t.Fatalf("expected alternative workouts")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`
Expected: FAIL（字段不存在）

**Step 3: Write minimal implementation**

- 在 `recommendation/types.go` 增加：

```go
type AlternativeWorkout struct {
	Type        string
	Title       string
	DurationMin int
	Intensity   string
	Tips        []string
}

type RecommendationOutput struct {
	...
	AlternativeWorkouts []AlternativeWorkout
}
```

- 在 `ai/recommender.go` 的 `RecommendationOutput` 增加 json 字段 `alternative_workouts`。
- 在 `processor.go` 的 `convertOutput` 中映射该字段。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/recommendation/types.go backend/internal/ai/recommender.go backend/internal/recommendation/processor.go backend/internal/recommendation/processor_test.go
git commit -m "推荐输出新增替代运动方案"
```

---

### Task 2: 规则兜底生成替代方案

**Files:**
- Modify: `backend/internal/recommendation/rules.go`
- Modify: `backend/internal/recommendation/rules_test.go`

**Step 1: Write the failing test**

在 `rules_test.go` 新增：

```go
func TestApplyRules_AddsAlternativeWorkoutsOnOverride(t *testing.T) {
	out := RecommendationOutput{ShouldRun: true}
	res := ApplyRules(RuleInput{WeatherRisk: "red"}, out)
	if len(res.Output.AlternativeWorkouts) == 0 {
		t.Fatalf("expected alternative workouts")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`
Expected: FAIL

**Step 3: Write minimal implementation**

在 `ApplyRules` 中：
- 当 override 触发时，如果 `AlternativeWorkouts` 为空，填入默认三条替代方案（跑步机/力量/拉伸）。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/recommendation/rules.go backend/internal/recommendation/rules_test.go
git commit -m "安全降级补齐替代运动建议"
```

---

### Task 3: 输出 JSON 结构验证

**Files:**
- Modify: `backend/internal/recommendation/processor_test.go`

**Step 1: Write the failing test**

在 `processor_test.go` 中将 `RecommendationOutput` 输出 JSON 解析，验证包含 `alternative_workouts` 字段。

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`
Expected: FAIL

**Step 3: Write minimal implementation**

若仍失败，补齐 `json` tag 或转换逻辑。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/recommendation/processor_test.go

git commit -m "推荐输出JSON包含替代方案"
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

补充 `alternative_workouts` schema 字段说明。

**Step 4: Run test to verify it passes**

无

**Step 5: Commit**

```bash
git add docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md

git commit -m "文档补充替代运动建议字段"
```

