# 训练反馈影响建议 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 推荐生成时读取最近一条训练反馈（log/activity），仅当有自由文本时将“自由文本 + 对应训练总结结构化字段”写入 AI 输入。

**Architecture:** 在 recommendation 生成流程中新增查询训练反馈与训练总结并注入 AI 输入；不新增存储结构，仅扩展推荐输入结构体与测试。

**Tech Stack:** Go、Kratos HTTP、PostgreSQL、OpenAI

---

### Task 1: 扩展 AI 输入结构与推荐输入模型（TDD）

**Files:**
- Modify: `backend/internal/ai/recommender.go`
- Modify: `backend/internal/recommendation/processor.go`
- Test: `backend/internal/recommendation/processor_test.go`

**Step 1: Write the failing test**

在 `processor_test.go` 新增断言：当存在反馈文本时，输入 JSON 包含 `latest_training_feedback`。

示例：
```go
func TestGenerateRecommendation_IncludesLatestTrainingFeedback(t *testing.T) {
	store := &fakeStore{
		profile: storage.UserProfile{UserID: "u1", LocationLat: 1, LocationLng: 2, Country: "CN", Province: "SH", City: "SH", AbilityLevel: "beginner"},
		latestFeedback: storage.TrainingFeedback{UserID: "u1", SourceType: "log", SourceID: "log-1", Content: "太累了"},
		latestSummary: storage.TrainingSummary{SummaryID: "s1", UserID: "u1", SourceType: "log", SourceID: "log-1", CompletionRate: "ok"},
	}
	p := NewProcessor(store, safeWeather{}, fakeAI{})
	if _, err := p.Generate(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	var input ai.RecommendationInput
	_ = json.Unmarshal(store.lastRec.InputJSON, &input)
	if input.LatestTrainingFeedback == nil {
		t.Fatalf("expected latest_training_feedback")
	}
	if input.LatestTrainingFeedback.Content == "" {
		t.Fatalf("expected feedback content")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/recommendation -v`  
Expected: FAIL（字段不存在）

**Step 3: Write minimal implementation**

- `ai.RecommendationInput` 增加 `LatestTrainingFeedback *RecommendationTrainingFeedback`。
- 新增结构体：
  - `RecommendationTrainingFeedback { SourceType, SourceID, CreatedAt, Content, Summary }`
  - `RecommendationTrainingSummary { CompletionRate, IntensityMatch, RecoveryAdvice, AnomalyNotes, PerformanceNotes, NextSuggestion }`
- `recommendation.Processor.Generate`：
  - 新增 `store.GetLatestTrainingFeedback(userID)` 方法（仅返回 `content` 非空者）。
  - 若存在，查询 `store.GetTrainingSummaryBySource(source_type, source_id)`。
  - 组装并写入 `input.LatestTrainingFeedback`。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/recommendation -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/ai/recommender.go backend/internal/recommendation/processor.go backend/internal/recommendation/processor_test.go

git commit -m "推荐输入引入最近训练反馈"
```

---

### Task 2: Storage 支持获取最近训练反馈（TDD）

**Files:**
- Modify: `backend/internal/storage/postgres.go`
- Test: `backend/internal/storage/postgres_baseline_test.go`

**Step 1: Write the failing test**

在 `postgres_baseline_test.go` 增加：

```go
func TestGetLatestTrainingFeedback(t *testing.T) {
	store := setupTestStore(t)
	feedback := TrainingFeedback{
		FeedbackID: "f1", UserID: "u1", SourceType: "log", SourceID: "log-1", Content: "太累了",
		CreatedAt: time.Now(),
	}
	if err := store.CreateTrainingFeedback(context.Background(), feedback); err != nil {
		t.Fatalf("create feedback failed: %v", err)
	}
	got, err := store.GetLatestTrainingFeedback(context.Background(), "u1")
	if err != nil {
		t.Fatalf("get latest feedback failed: %v", err)
	}
	if got.Content == "" {
		t.Fatalf("expected content")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./backend/internal/storage -v`  
Expected: FAIL（方法不存在）

**Step 3: Write minimal implementation**

- `PostgresStore` 新增方法：
  - `GetLatestTrainingFeedback(ctx, userID) (TrainingFeedback, error)`
  - SQL：按 `created_at desc` 取 1 条，要求 `content <> ''`，`deleted_at IS NULL`
- `Store` 接口在 `recommendation/processor.go` 中添加此方法。

**Step 4: Run test to verify it passes**

Run: `go test ./backend/internal/storage -v`  
Expected: PASS（或 SKIP if DSN not set）

**Step 5: Commit**

```bash
git add backend/internal/storage/postgres.go backend/internal/storage/postgres_baseline_test.go

git commit -m "新增获取最近训练反馈"
```

---

### Task 3: 文档同步

**Files:**
- Modify: `docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md`

**Step 1: Update AI Input Schema**

- `AIRecommendationInput` 增加 `latest_training_feedback` 字段说明。

**Step 2: Commit**

```bash
git add docs/plans/2026-03-09-stridewise-openapi-and-json-schema.md

git commit -m "文档补充训练反馈输入"
```

---

### Task 4: 全量回归与飞行记录

**Step 1: Run full tests**

Run: `go test ./backend/...`  
Expected: PASS

**Step 2: Flight Recorder**

```bash
python3 /Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py "Feature" "推荐输入加入最近训练反馈" "影响推荐输入结构，可能引起模型输出变化；已回归测试" "S2"
```

**Step 3: Commit**

```bash
git add docs/AI_CHANGELOG.md

git commit -m "记录训练反馈影响建议变更"
```
