# Keep Raw Track/HR Write Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extend Keep live connector to persist raw `geoPoints`/`heartRates` both original and decoded into `RawActivity.Raw` while keeping running_page decoding behavior.

**Architecture:** Reuse existing Keep decoder. Store original payload fields and decoded arrays in Raw. Keep polyline generation and GCJ02->WGS84 transform consistent. Tests drive Raw field expectations.

**Tech Stack:** Go, Kratos, Asynq, PostgreSQL

---

### Task 1: Raw 写入测试（TDD）

**Files:**
- Modify: `backend/internal/connector/keep/connector_test.go`

**Step 1: Write the failing test**

```go
func TestConnector_FetchActivities_WritesRawDecoded(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/v1.1/users/login", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"data":{"token":"t1"}}`))
    })
    mux.HandleFunc("/pd/v3/stats/detail", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"data":{"records":[{"logs":[{"stats":{"id":"run_1","isDoubtful":false}}]}],"lastTimestamp":0}}`))
    })

    geoPayload, _ := encodeRunmapForTest([]map[string]any{{"latitude": 30.0, "longitude": 120.0, "timestamp": 1}})
    hrPayload, _ := encodeRunmapForTest([]map[string]any{{"timestamp": 1, "hr": 90}})

    mux.HandleFunc("/pd/v3/runninglog/run_1", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"data":{"id":"abc_1","startTime":1700000000000,"endTime":1700000600000,"duration":600,"distance":1000,"dataType":"outdoorRunning","timezone":"Asia/Shanghai","geoPoints":"` + geoPayload + `","heartRate":{"averageHeartRate":88,"heartRates":"` + hrPayload + `"}}}`))
    })
    srv := httptest.NewServer(mux)
    t.Cleanup(srv.Close)

    c := NewLive("13000000000", "pass", srv.URL, srv.Client())
    res, err := c.FetchActivities(context.Background(), "u1", syncjob.Checkpoint{})
    if err != nil {
        t.Fatalf("fetch err: %v", err)
    }
    if len(res.Activities) != 1 {
        t.Fatalf("expected 1 activity, got %d", len(res.Activities))
    }
    raw := res.Activities[0].Raw
    if raw["geo_points_raw"] == "" {
        t.Fatal("expected geo_points_raw")
    }
    if raw["geo_points_decoded"] == nil {
        t.Fatal("expected geo_points_decoded")
    }
    if raw["heart_rate_raw"] == nil {
        t.Fatal("expected heart_rate_raw")
    }
    if raw["heart_rates_decoded"] == nil {
        t.Fatal("expected heart_rates_decoded")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/connector/keep -run TestConnector_FetchActivities_WritesRawDecoded -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- 在 `parseKeepRunData` 中写入:
  - `geo_points_raw`（原始 geoPoints 字符串）
  - `geo_points_decoded`（解码结果数组）
  - `heart_rate_raw`（原始 heartRate 对象）
  - `heart_rates_decoded`（解码结果数组）
- 解码失败时不写 decoded 字段（保持 nil）。

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/connector/keep -run TestConnector_FetchActivities_WritesRawDecoded -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/connector/keep/connector.go backend/internal/connector/keep/connector_test.go
git commit -m "feat: write keep raw decoded points"
```

**Step 6: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Feature" "Keep Raw 写入 geoPoints/heartRates 原文与解码结果" "风险在 Raw 体积增大与序列化性能下降" "S2"
```

---

### Task 2: 回归

**Step 1: Run full test suite**

Run: `cd backend && go test ./...`
Expected: PASS

**Step 2: Commit (if needed)**

```bash
git add <files>
git commit -m "fix: stabilize keep raw write"
```

**Step 3: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Bugfix" "Keep Raw 写入回归修复" "风险低，主要为补丁级修复" "S3"
```
