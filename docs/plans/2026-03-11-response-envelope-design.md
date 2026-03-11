# 异常兜底标准化响应设计 v1.0.0

> **版本记录**
> - v1.0.0 (2026-03-11)：首版，统一响应包裹并标准化兜底元信息与错误结构。

## 1. 目标与范围

**目标**
- 内部 API 全部返回统一响应包裹结构（成功与错误一致）。
- 异常兜底信息标准化，便于追溯与统计。
- 不改变业务逻辑，仅改变响应结构与错误表达方式。

**范围**
- `/internal/**` 全部接口（含 health/metrics）。
- 推荐、训练、天气、同步、基线、反馈等所有内部链路。

**非目标**
- 不新增外部 API。
- 不引入新的外部依赖。
- 不调整业务规则与核心计算逻辑。

## 2. 统一响应结构

所有响应统一为：

```json
{
  "data": {},
  "error": null,
  "meta": {
    "request_id": "uuid",
    "timestamp": "2026-03-11T12:34:56Z",
    "fallback_reason": "weather_api_failed",
    "confidence": 0.4
  }
}
```

字段说明：
- `data`: 成功时为业务数据；失败时为 `null`。
- `error`: 成功时为 `null`；失败时包含 `code/message`。
- `meta.request_id`: 服务端生成或透传请求头。
- `meta.timestamp`: 服务端响应时间（RFC3339）。
- `meta.fallback_reason`: 触发兜底时填写，否则为空。
- `meta.confidence`: 默认 `1.0`，兜底时降低。

## 3. 错误结构与错误码

错误结构：

```json
"error": { "code": "bad_request", "message": "user_id required" }
```

最小错误码集合：
- `bad_request` (400)
- `unauthorized` (401)
- `forbidden` (403)
- `not_found` (404)
- `conflict` (409)
- `dependency_unavailable` (503)
- `internal_error` (500)

错误消息保持现有可读文本。

## 4. 兜底原因枚举（fallback_reason）

- `insufficient_data`
- `weather_api_failed`
- `ai_unavailable`
- `ability_level_not_ready`
- `safety_override`
- `third_party_import_failed`
- `field_missing`

说明：
- 非兜底场景可为空。
- `safety_override` 仅在安全规则触发强制降级时使用。

## 5. 实现策略

- 将 `writeJSON` 替换为统一封装：`writeSuccess`/`writeError`。
- `http.Error` 统一替换为 `writeError`，保持原 HTTP 状态码。
- 通过 helper 生成 `meta`，默认 `confidence=1.0`。
- 推荐、天气等已存在兜底逻辑的接口，填充 `fallback_reason/confidence`。

## 6. 测试策略

- Server 测试：所有典型接口响应包含 `data/error/meta`。
- Error 测试：错误响应仍为 JSON 包裹且 `error.code` 正确。
- 回归测试：`go test ./backend/...`。

## 7. 风险与回滚

- 风险：响应结构变化可能影响调用方与测试断言。
- 回滚：恢复原 `writeJSON/http.Error` 行为即可。
