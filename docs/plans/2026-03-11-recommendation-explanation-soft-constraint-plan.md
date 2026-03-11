# 训练建议解释数量软约束 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 移除 `explanation` 最小条数硬校验与 Schema `minItems` 约束，允许 AI 少于 2 条解释也直接返回。

**Architecture:** 仅调整 AI 输出校验与文档约束，不新增数据结构与接口。以 TDD 方式先写单测，再修改校验与文档。

**Tech Stack:** Go, pgx, OpenAPI/JSON Schema Markdown, standard Go tests

---

### Task 1: 放宽 AI 输出解释条数校验（TDD）

**Files:**
- Modify: `backend/internal/ai/openai_recommender.go`
- Modify: `backend/internal/recommendation/processor_test.go`

**Step 1: Write the failing test**

在 `backend/internal/recommendation/processor_test.go` 新增用例，模拟 AI 返回 `explanation` 仅 1 条，期望仍成功落库并返回。

```go
func TestGenerateRecommendation_AIExplanationShortStillAccepted(t *testing.T) {
    store := &fakeStore{ /* 同已有 AI 测试初始化 */ }
    p := NewProcessor(store, safeWeather{}, fakeAIWithShortExplanation{})
    if _, err := p.Generate(context.Background(), "u1"); err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -run TestGenerateRecommendation_AIExplanationShortStillAccepted -v`
Expected: FAIL with `explanation requires at least 2 items`

**Step 3: Write minimal implementation**

在 `backend/internal/ai/openai_recommender.go` 的 `validateRecommendationOutput` 中移除 `len(out.Explanation) < 2` 校验。

```go
// 删除 explanation 数量校验，仅保留关键字段必填
```

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -run TestGenerateRecommendation_AIExplanationShortStillAccepted -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/ai/openai_recommender.go backend/internal/recommendation/processor_test.go
git commit -m "放宽AI解释条数校验"
```

### Task 2: 更新文档与 Schema

**Files:**
- Modify: `request.md`
- Modify: `docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`

**Step 1: Update request.md**

将“解释≥2”改为“期望≥2，允许不足”。

**Step 2: Update OpenAPI/JSON Schema**

移除 `explanation` 的 `minItems` 约束。

**Step 3: Commit**

```bash
git add request.md docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md
git commit -m "更新解释条数软约束文档"
```

### Task 3: 回归测试

**Files:**
- Test: `backend/internal/ai`
- Test: `backend/internal/recommendation`

**Step 1: Run tests**

Run: `go test ./backend/internal/ai ./backend/internal/recommendation -v`
Expected: PASS

**Step 2: Commit (if any fixes required)**

```bash
git add -A
git commit -m "修复解释条数软约束回归测试"
```
