# Keep Reverse Login Import Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement Keep reverse-login ingestion (running_page-compatible) using phone/password from config, with incremental sync and standardization.

**Architecture:** Replace Keep offline connector with a live connector that logs in to Keep, fetches activity IDs and detail payloads, decodes run maps and heart-rate data, generates polyline, and maps into `RawActivity`. Use checkpoint for incremental sync and keep the rest of the pipeline unchanged.

**Tech Stack:** Go, net/http, Kratos, Asynq, PostgreSQL

---

### Task 1: Keep API client (TDD)

**Files:**
- Create: `backend/internal/connector/keep/client.go`
- Create: `backend/internal/connector/keep/client_test.go`

**Step 1: Write the failing test**

```go
package keep

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestKeepClient_LoginAndFetchList(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/v1.1/users/login", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"data":{"token":"t1"}}`))
    })
    mux.HandleFunc("/pd/v3/stats/detail", func(w http.ResponseWriter, r *http.Request) {
        auth := r.Header.Get("Authorization")
        if auth != "Bearer t1" {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"data":{"records":[{"logs":[{"stats":{"id":"run_1","isDoubtful":false}}]}],"lastTimestamp":0}}`))
    })
    srv := httptest.NewServer(mux)
    t.Cleanup(srv.Close)

    client := NewKeepClient(srv.URL, nil)
    token, err := client.Login(context.Background(), "13000000000", "pass")
    if err != nil {
        t.Fatalf("login err: %v", err)
    }
    if token != "t1" {
        t.Fatalf("unexpected token: %s", token)
    }

    ids, last, err := client.FetchRunIDs(context.Background(), "t1", "running", 0)
    if err != nil {
        t.Fatalf("fetch ids err: %v", err)
    }
    if last != 0 {
        t.Fatalf("expected lastTimestamp 0, got %d", last)
    }
    if len(ids) != 1 || ids[0] != "run_1" {
        t.Fatalf("unexpected ids: %+v", ids)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/connector/keep -run TestKeepClient_LoginAndFetchList -v`
Expected: FAIL (client not implemented)

**Step 3: Write minimal implementation**

```go
// client.go
package keep

import (
    "context"
    "encoding/json"
    "errors"
    "net/http"
    "net/url"
    "strconv"
)

type KeepClient struct {
    baseURL string
    httpClient *http.Client
}

func NewKeepClient(baseURL string, httpClient *http.Client) *KeepClient {
    if httpClient == nil {
        httpClient = http.DefaultClient
    }
    return &KeepClient{baseURL: baseURL, httpClient: httpClient}
}

func (c *KeepClient) Login(ctx context.Context, phone, password string) (string, error) {
    if phone == "" || password == "" {
        return "", errors.New("keep credential is empty")
    }
    form := url.Values{}
    form.Set("mobile", phone)
    form.Set("password", password)

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1.1/users/login", strings.NewReader(form.Encode()))
    if err != nil {
        return "", err
    }
    req.Header.Set("User-Agent", keepUserAgent)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return "", errors.New("keep login failed")
    }

    var body struct {
        Data struct {
            Token string `json:"token"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
        return "", err
    }
    if body.Data.Token == "" {
        return "", errors.New("keep token empty")
    }
    return body.Data.Token, nil
}

func (c *KeepClient) FetchRunIDs(ctx context.Context, token, sportType string, lastDate int64) ([]string, int64, error) {
    q := url.Values{}
    q.Set("dateUnit", "all")
    q.Set("type", sportType)
    q.Set("lastDate", strconv.FormatInt(lastDate, 10))

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/pd/v3/stats/detail?"+q.Encode(), nil)
    if err != nil {
        return nil, 0, err
    }
    req.Header.Set("User-Agent", keepUserAgent)
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, 0, err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, 0, errors.New("keep fetch list failed")
    }

    var body struct {
        Data struct {
            Records []struct {
                Logs []struct {
                    Stats struct {
                        ID         string `json:"id"`
                        IsDoubtful bool   `json:"isDoubtful"`
                    } `json:"stats"`
                } `json:"logs"`
            } `json:"records"`
            LastTimestamp int64 `json:"lastTimestamp"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
        return nil, 0, err
    }

    ids := make([]string, 0)
    for _, record := range body.Data.Records {
        for _, log := range record.Logs {
            if !log.Stats.IsDoubtful {
                ids = append(ids, log.Stats.ID)
            }
        }
    }

    return ids, body.Data.LastTimestamp, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/connector/keep -run TestKeepClient_LoginAndFetchList -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/connector/keep/client.go backend/internal/connector/keep/client_test.go
git commit -m "feat: add keep api client"
```

**Step 6: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Feature" "新增 Keep 逆向登录 API 客户端" "风险在接口返回结构变化导致解析失败" "S2"
```

---

### Task 2: Keep 逆向登录连接器（TDD）

**Files:**
- Modify: `backend/internal/connector/keep/connector.go`
- Modify: `backend/internal/connector/keep/connector_test.go`

**Step 1: Write the failing test**

```go
package keep

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    syncjob "stridewise/backend/internal/sync"
)

func TestConnector_FetchActivities_Live(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/v1.1/users/login", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"data":{"token":"t1"}}`))
    })
    mux.HandleFunc("/pd/v3/stats/detail", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"data":{"records":[{"logs":[{"stats":{"id":"run_1","isDoubtful":false}}]}],"lastTimestamp":0}}`))
    })
    mux.HandleFunc("/pd/v3/runninglog/run_1", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"data":{"id":"abc_1","startTime":1700000000000,"endTime":1700000600000,"duration":600,"distance":1000,"dataType":"outdoorRunning","timezone":"Asia/Shanghai","geoPoints":null,"heartRate":null}}`))
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
    if res.Activities[0].SourceActivityID != "1" {
        t.Fatalf("unexpected id: %s", res.Activities[0].SourceActivityID)
    }
    if res.Activities[0].MovingTimeSec != 600 {
        t.Fatalf("unexpected moving time: %d", res.Activities[0].MovingTimeSec)
    }
    if res.LastSyncedAt.IsZero() {
        t.Fatal("expected last synced at")
    }
    if res.Activities[0].StartTime.After(time.Now().Add(24 * time.Hour)) {
        t.Fatal("unexpected start time")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/connector/keep -run TestConnector_FetchActivities_Live -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- 新增 `NewLive(phone, password, baseURL, httpClient)`。
- Keep connector 走 live 拉取逻辑：
  - `Login` -> `FetchRunIDs` -> `FetchRunDetail` -> 组装 `RawActivity`。
- 参考 running_page：
  - `KEEP_SPORT_TYPES = ["running","hiking","cycling"]`
  - `dataType` 映射到名称（`KEEP2STRAVA`）
  - `startTime`/`duration`/`distance` 使用 `run_data` 字段
  - 若 `geoPoints` 为空：`summary_polyline` 为空
  - `sleep(1)` 频控

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/connector/keep -run TestConnector_FetchActivities_Live -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/connector/keep/connector.go backend/internal/connector/keep/connector_test.go
git commit -m "feat: keep live connector"
```

**Step 6: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Feature" "实现 Keep 逆向登录采集连接器" "风险在 Keep API 变更或封禁导致不可用" "S2"
```

---

### Task 3: Keep runmap 解码与 polyline（TDD）

**Files:**
- Modify: `backend/internal/connector/keep/connector.go`
- Create: `backend/internal/connector/keep/decoder_test.go`

**Step 1: Write the failing test**

```go
package keep

import (
    "testing"
)

func TestDecodeRunmapData_RoundTrip(t *testing.T) {
    points := []map[string]any{{"latitude": 30.0, "longitude": 120.0, "timestamp": 1}}
    payload, err := encodeRunmapForTest(points)
    if err != nil {
        t.Fatalf("encode: %v", err)
    }

    got, err := decodeRunmapData(payload, false)
    if err != nil {
        t.Fatalf("decode: %v", err)
    }
    if len(got) != 1 {
        t.Fatalf("expected 1 point, got %d", len(got))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/connector/keep -run TestDecodeRunmapData_RoundTrip -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- 添加 `decodeRunmapData`（base64 -> AES‑CBC -> gzip/zlib 解压 -> JSON）。
- 为测试实现 `encodeRunmapForTest`（仅测试文件使用）。
- 用 `github.com/twpayne/go-polyline` 生成 `summary_polyline`。
- GCJ02->WGS84 转换：实现 `gcj2wgs` 函数（标准算法）。

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/connector/keep -run TestDecodeRunmapData_RoundTrip -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/connector/keep/connector.go backend/internal/connector/keep/decoder_test.go backend/go.mod backend/go.sum
git commit -m "feat: keep runmap decode and polyline"
```

**Step 6: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Feature" "实现 Keep runmap 解码与 polyline 生成" "风险在解码逻辑偏差导致轨迹丢失" "S2"
```

---

### Task 4: 配置与 worker 接入（TDD）

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/config/config.yaml`
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/cmd/worker/main.go`

**Step 1: Write the failing test**

```go
package config

import "testing"

func TestLoad_KeepCredentials(t *testing.T) {
    // 复用现有 config_test，新增断言即可
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/config -v`
Expected: FAIL

**Step 3: Write minimal implementation**

- 配置新增 `keep.phone_number` / `keep.password`。
- Worker 使用 `NewLive(cfg.Keep.PhoneNumber, cfg.Keep.Password, defaultBaseURL, nil)`。

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/config -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/config/config.go backend/internal/config/config_test.go backend/config/config.yaml backend/cmd/worker/main.go
git commit -m "feat: keep live credentials in config"
```

**Step 6: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Feature" "配置 Keep 逆向登录凭证并接入 worker" "风险在明文凭证泄露；需限制配置访问" "S2"
```

---

### Task 5: 全量回归

**Step 1: Run full test suite**

Run: `cd backend && go test ./...`
Expected: PASS

**Step 2: Commit (if needed)**

```bash
git add <files>
git commit -m "fix: stabilize keep live import"
```

**Step 3: Flight recorder**

```bash
python3 "/Users/zhangyukun/.codex/skills/flight-recorder/scripts/log_change.py" "Bugfix" "Keep 逆向登录回归修复" "风险低，主要为补丁级修复" "S3"
```
