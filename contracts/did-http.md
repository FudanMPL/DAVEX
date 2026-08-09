# DAVEX DID 后端 HTTP 契约（第一阶段）

## 1. 范围

本文档定义 DID 新功能在以下链路中的 HTTP/JSON 契约：

```text
DAVEX 前端
    -> DAVEX_center 或 DAVEX_agent
    -> DAVEX_base DidService/DidBackendClient
    -> did/backend
    -> sdk-go
    -> ChainMaker simple-did 合约
```

本契约只为 DAVEX 新增 DID 能力，不改变任何现有 DAVEX 业务接口、JWT、ABAC 或 center-agent 直接 URL 交互方式。

## 2. 服务与配置

### 2.1 DAVEX Java 服务

- Center 当前端口：`9900`
- Agent 当前端口：`8080`
- DID 公开 API 前缀：`/api/v1/did`

### 2.2 Go DID backend

- 服务目录：`did/backend`
- DAVEX 集成默认地址：`http://localhost:8081`
- 公开健康检查：`GET /api/health`
- Swagger UI：`GET /swagger/`（原始契约：`GET /swagger/openapi.yaml`）
- 其他接口：`/api/**`，需要 Bearer token

Java 配置：

```yaml
did:
  enabled: ${DID_ENABLED:false}
  backend-url: ${DID_BACKEND_URL:http://localhost:8081}
  connect-timeout-ms: ${DID_CONNECT_TIMEOUT_MS:2000}
  request-timeout-ms: ${DID_REQUEST_TIMEOUT_MS:5000}
  token: ${DID_BACKEND_TOKEN:}
```

`did.enabled=false` 或 Go backend 不可用时，只允许 DID 新接口返回错误，不得影响 center/agent 启动和原有功能。

## 3. 身份传递

### 3.1 前端到 Java

DID 接口可选接收：

```http
X-DID-Actor: <actor-alias>
X-Request-ID: <trace-id>
```

- `X-DID-Actor` 表示希望使用的 DID actor alias。
- 它不代表已获授权，Java 只将它作为 `DidActorContext` 的候选 actor。
- 普通调用方能否切换 actor，仍由 Go backend 的 token 权限与智能合约决定。
- 前端不传递 Go backend Bearer token。

### 3.2 Java 到 Go backend

Java 由本地配置添加：

```http
Authorization: Bearer <DID_BACKEND_TOKEN>
X-Actor-Alias: <DidActorContext.actorAlias>
X-Request-ID: <DidActorContext.requestId>
Content-Type: application/json; charset=utf-8
```

`GET /api/health` 是例外，不需要 Bearer token。

## 4. DAVEX 统一响应

Java 对前端统一返回：

```json
{
  "code": 0,
  "message": "ok",
  "data": {},
  "requestId": "20260808T120000.000000000",
  "tx": {
    "txId": "...",
    "blockHeight": 1,
    "gasUsed": 0
  },
  "upstreamCode": "OK"
}
```

约定：

- `code == 0`：成功。
- `data`：Go backend 响应中的 `data`。
- `requestId`：优先使用 Go backend 返回的 `requestId`，否则使用 Java 本次请求 ID。
- `tx`：存在链上交易时透传 Go backend 的交易摘要。
- `upstreamCode`：保留 Go backend 的字符串错误码，便于定位。

失败示例：

```json
{
  "code": 41004,
  "message": "DID backend 不可用",
  "data": null,
  "requestId": "java-generated-request-id",
  "tx": null,
  "upstreamCode": null
}
```

### 4.1 Java 错误码

| code | 含义 |
|---:|---|
| `0` | 成功 |
| `41001` | DID 功能未启用 |
| `41002` | Java 侧 DID 请求参数不完整 |
| `41003` | Go backend 返回了无效 JSON/无效协议 |
| `41004` | Go backend 不可达、超时或连接中断 |
| `41005` | Go backend 拒绝了请求，具体原因见 `upstreamCode` 和 `message` |

## 5. 第一阶段接口映射

| DAVEX Java 接口 | Go backend 接口 | 是否需要 actor |
|---|---|---|
| `GET /api/v1/did/health` | `GET /api/health` | 否 |
| `GET /api/v1/did/ready` | `GET /api/ready` | 是 |
| `GET /api/v1/did/session` | `GET /api/system/session` | 是 |
| `GET /api/v1/did/status` | `GET /api/system/status` | 是 |
| `POST /api/v1/did/governance` | `POST /api/system/governance` | 是 |
| `PUT /api/v1/did/roles` | `PUT /api/system/roles` | 是 |
| `GET /api/v1/did/roles?did=...` | `GET /api/system/roles?did=...` | 是 |
| `POST /api/v1/did/identity/generate` | `POST /api/did/generateDID` | 是 |
| `POST /api/v1/did/identity/register` | `POST /api/did/registerDID` | 是 |
| `GET /api/v1/did/identity/query?did=...` | `GET /api/did/query?did=...` | 是 |
| `POST /api/v1/did/policy/register` | `POST /api/policy/register` | 是 |
| `GET /api/v1/did/policy/query?policyID=...` | `GET /api/policy/query?policyID=...` | 是 |
| `POST /api/v1/did/policy/deactivate` | `POST /api/policy/deactivate` | 是 |
| `POST /api/v1/did/credential/issue` | `POST /api/vc/issue` | 是 |
| `GET /api/v1/did/credential/query?vcID=...` | `GET /api/vc/query?vcID=...` | 是 |
| `POST /api/v1/did/credential/verify` | `POST /api/vc/verify` | 是 |
| `POST /api/v1/did/presentation/nonce` | `POST /api/vp/nonce` | 是 |
| `POST /api/v1/did/presentation/generate` | `POST /api/vp/generate` | 是 |
| `POST /api/v1/did/presentation/verify` | `POST /api/vp/verify` | 是 |

center 和 agent 均可暴露上述同名接口。是否允许执行由 actor 及链上角色决定，不由 center/agent 类型决定。

## 6. 请求定义

### 6.1 健康、就绪、会话和状态

#### `GET /api/v1/did/health`

无请求体，不需要 actor。只检查 Go HTTP 服务。

#### `GET /api/v1/did/ready`

无请求体。检查 Go backend、ChainMaker 和合约是否就绪。

#### `GET /api/v1/did/session`

无请求体。返回登录 actor、当前执行 actor 和可见 actor 列表。

#### `GET /api/v1/did/status`

无请求体。返回链、合约和 actor 状态。

### 6.2 治理与链上角色

绑定治理 DID：

```http
POST /api/v1/did/governance
```

```json
{
  "did": "did:gov:governance"
}
```

更新 DID 角色：

```http
PUT /api/v1/did/roles
```

```json
{
  "did": "did:gov:issuer",
  "roles": ["policy_manager", "credential_issuer"]
}
```

查询角色：

```http
GET /api/v1/did/roles?did=did:gov:issuer
```

`did` 必填。

### 6.3 生成 DID

```http
POST /api/v1/did/identity/generate
Content-Type: application/json
```

```json
{
  "did": "did:gov:issuer",
  "document": "{\"id\":\"did:gov:issuer\",\"name\":\"法院\"}"
}
```

该接口本地生成 DID 协议密钥和 DID 文档哈希，不上链。私钥仅保存在 Go backend 本地状态中，不经 Java 和前端返回。

### 6.4 注册 DID

```http
POST /api/v1/did/identity/register
Content-Type: application/json
```

```json
{
  "did": "did:gov:issuer"
}
```

将已在 Go backend 本地生成的 DID 注册上链。当前 actor 必须与 DID owner 的 ChainMaker Origin 一致。

### 6.5 查询 DID

```http
GET /api/v1/did/identity/query?did=did:gov:issuer
```

`did` 必填。返回链上 registry；如当前 Go backend 拥有完整文档，可同时返回本地文档。

### 6.6 策略

注册策略：

```http
POST /api/v1/did/policy/register
```

```json
{
  "issuerDID": "did:gov:issuer",
  "permissions": [
    {
      "policyID": "policy-case-read",
      "deptRole": "court",
      "authScope": "case",
      "dataLevel": "internal",
      "actionSet": ["read"],
      "validFrom": 1785900000,
      "validUntil": 1786000000
    }
  ]
}
```

查询策略：

```http
GET /api/v1/did/policy/query?policyID=policy-case-read
```

停用策略：

```http
POST /api/v1/did/policy/deactivate
```

```json
{
  "id": "policy-case-read"
}
```

### 6.7 签发 VC

```http
POST /api/v1/did/credential/issue
Content-Type: application/json
```

```json
{
  "vcID": "vc-case-read-001",
  "holderDID": "did:gov:holder",
  "issuerDID": "did:gov:issuer",
  "policyID": "policy-case-read",
  "entryIndex": 0,
  "expiresAt": 1786000000,
  "anonymousEligible": true
}
```

前置条件：

- issuer DID 已在当前 Go backend 生成并上链。
- 对应 policy 已在 Go backend 本地存在并已上链。
- issuer 拥有链上 `credential_issuer` 角色。

### 6.8 查询与验证 VC

查询：

```http
GET /api/v1/did/credential/query?vcID=vc-case-read-001
```

验证：

```http
POST /api/v1/did/credential/verify
Content-Type: application/json
```

```json
{
  "id": "vc-case-read-001"
}
```

### 6.9 签发一次性 nonce

```http
POST /api/v1/did/presentation/nonce
Content-Type: application/json
```

```json
{
  "verifierDID": "did:gov:verifier",
  "purpose": "case-read",
  "ttlSeconds": 120
}
```

### 6.10 生成 VP

```http
POST /api/v1/did/presentation/generate
Content-Type: application/json
```

```json
{
  "holderDID": "did:gov:holder",
  "verifierDID": "did:gov:verifier",
  "nonce": "nonce-...",
  "purpose": "case-read",
  "vcIDs": ["vc-case-read-001"]
}
```

### 6.11 验证 VP

```http
POST /api/v1/did/presentation/verify
Content-Type: application/json
```

```json
{
  "vp": {
    "holderDID": "did:gov:holder",
    "verifierDID": "did:gov:verifier",
    "nonce": "nonce-...",
    "purpose": "case-read",
    "credentials": [],
    "signature": "..."
  },
  "access": {
    "deptRole": "court",
    "authScope": "case",
    "dataLevel": "internal",
    "action": "read",
    "atTime": 1785950000
  }
}
```

`vp` 精确字段以 Go `protocol.StandardVP` 的 JSON 为准。Java 第一阶段不解析或重写密码学字段，只进行 JSON 透传和统一响应适配。

## 7. Go backend 原始响应

成功：

```json
{
  "code": "OK",
  "message": "success",
  "data": {},
  "requestId": "...",
  "tx": {}
}
```

失败：

```json
{
  "code": "CONTRACT_REJECTED",
  "message": "链上业务校验失败",
  "requestId": "..."
}
```

Java 适配规则：

- Go `code == "OK"`：Java `code = 0`。
- Go `code != "OK"`：Java `code = 41005`，`upstreamCode` 保留 Go code。
- Go 返回非 JSON：Java `code = 41003`。
- 连接失败或超时：Java `code = 41004`。

## 8. 隔离性验收

实现完成后必须手动验证：

1. `DID_ENABLED=false` 时，center 和 agent 可正常启动。
2. 不启动 Go DID backend 时，center 和 agent 可正常启动。
3. DID 接口在关闭时返回 `41001`，不影响其他 Controller。
4. DID 开启但 Go backend 不可达时，DID 接口返回 `41004`。
5. Go backend 可达时，`health` 完成 Java -> Go 返回。
6. 配置有效 actor 和 ChainMaker 环境时，完成 DID generate -> register -> query 闭环。
7. 本阶段不修改现有 JWT、ABAC、目录、文件、查询、MPC、TEE 和 verdict 业务。
