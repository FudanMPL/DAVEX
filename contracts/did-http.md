# DAVEX DID 后端 HTTP 契约

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

- Center 当前本地端口：`9999`（以实际运行配置为准）
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

### 5.1 匿名认证与撤销增量映射（本阶段由 Center 暴露）

| DAVEX Center 接口 | Go backend 接口 | 执行 actor |
|---|---|---|
| `GET /api/v1/did/privacy/group?groupID=...` | `GET /api/privacy/group?groupID=...` | 任一已认证 actor |
| `POST /api/v1/did/privacy/group/member` | `POST /api/privacy/group/member` | 群组签发方 `issuer` |
| `POST /api/v1/did/privacy/presentation/generate` | `POST /api/privacy/vp/generate` | 持有者 `holder` |
| `POST /api/v1/did/privacy/presentation/verify` | `POST /api/privacy/vp/verify` | 验证方 `verifier` |
| `GET /api/v1/did/privacy/keyimage?value=...` | `GET /api/privacy/keyimage?value=...` | 任一已认证 actor |
| `GET /api/v1/did/revocation/issuer?eventType=...&issuerDID=...` | `GET /api/revocation/issuer` | 任一已认证 actor |
| `GET /api/v1/did/revocation/committee?groupID=...` | `GET /api/revocation/committee` | 任一已认证 actor |
| `POST /api/v1/did/revocation/requests` | `POST /api/revocation/requests` | 事件签发方 `bootstrap` |
| `GET /api/v1/did/revocation/requests/{draftID}` | `GET /api/revocation/requests/{draftID}` | 任一已认证 actor |
| `POST /api/v1/did/revocation/requests/{draftID}/approvals` | 同路径的 Go 接口 | 当前有效委员 `issuer`/`verifier` |
| `POST /api/v1/did/revocation/execute` | `POST /api/revocation/execute` | 演示由 `bootstrap` 执行 |
| `GET /api/v1/did/revocation/logs?vcID=...` | `GET /api/revocation/logs?vcID=...` | 任一已认证 actor |
| `GET /api/v1/did/revocation/consumed?hash=...` | `GET /api/revocation/consumed?hash=...` | 任一已认证 actor |

以上新接口不自动扩展旧 `/api/v1/control/**` 或 `DAVEX_agent` 路由。Java 只校验必填查询参数并转发 JSON，不生成签名、不代替委员批准。`X-DID-Actor` 仍是候选身份：本地演示使用管理员 Go token 切换不同 SDK actor，不能把下拉框视为生产授权。

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

### 6.12 匿名资格及匿名 VP

入群请求 `POST /api/v1/did/privacy/group/member`：`{"groupID":"group-...","vcID":"vc-..."}`。VC 必须有效、允许匿名且与群组策略相符；群组由受控初始化预先创建。`GET /privacy/group` 返回 `groupID`、`policyID`、`memberEpoch`、`lHash`、`status` 等链上数据。

匿名认证沿用 6.9 的 nonce，目的 `purpose` 必须一致。持有者生成请求：

```json
{"vpID":"pvp-...","groupID":"group-...","holderDID":"did:...:holder","verifierDID":"did:...:verifier","nonce":"nonce-...","purpose":"davex-anonymous"}
```

`POST /privacy/presentation/generate` 成功的 `data` 含 `vp` 和 `qualification`。`holderDID` 仅用于 Go 定位本地 LSAG 私钥，不得放进最终匿名 VP。验证方将两者原样回传，并加访问条件：

```json
{"vp":{},"qualification":{},"access":{"deptRole":"community_correction_officer","authScope":"community_correction","dataLevel":"restricted","action":"read","atTime":1791440000}}
```

`vp`、`qualification` 以生成接口实际返回对象为准。成功时 `data.verified=true` 且有 `data.keyImage`；`GET /privacy/keyimage?value=...` 返回 `data.used=true`。同一个 VP 重放应失败，验证结果不应泄露持有者 DID 或 VC ID。nonce 与密钥映像由现有合约消费，Java/前端不得本地伪造成功。

### 6.13 撤销申请、批准和执行

群组对应的事件签发方白名单与 2-of-2 委员会由受控演示准备创建，普通页面只查询其状态。`GET /revocation/issuer` 用 `eventType`、`issuerDID`，`GET /revocation/committee` 用 `groupID`。

`POST /api/v1/did/revocation/requests` 的请求字段：

```json
{"groupID":"group-...","vcID":"vc-...","eventIssuerDID":"did:...:governance","eventType":"event-...","scopeType":"group","scopeID":"group-..."}
```

`draftID` 可选，省略时由 Go 生成。成功返回 `data.id`、`data.status="Pending"`、`data.approvals`。页面中的事件类型就是已登记的撤销原因类型；补充说明不属于现有合约或 Go 草案字段，不得声称已持久化。`GET /revocation/requests/{draftID}` 返回真实草案；委员会 `threshold` 由 `GET /revocation/committee?groupID=...` 读取，批准数由草案 `approvals` 数量得出。

每位委员分别用自身 actor 调用 `POST /revocation/requests/{draftID}/approvals`，请求体 `{}`；同一委员不能重复批准。达到门限后调用 `POST /revocation/execute`，请求体 `{"draftID":"..."}`。这是不可逆链上操作，失败或超时应先查询草案、VC 和日志，不能盲目重试。成功返回 `data.vcProof.status="Revoked"`、`data.group`、`data.credentialHash`、`data.log`。最终再次查询 VC、群组、撤销日志和 consumed 状态；群组 epoch 应增长、`lHash` 应改变，已撤销 VC 不可再生成新的匿名 VP。

新增接口沿用 4.1 的 Java 错误码。常见 Go `upstreamCode` 包括 `MEMBER_BINDING_NOT_FOUND`、`CONTRACT_REJECTED`、`NOT_COMMITTEE_MEMBER`、`THRESHOLD_NOT_MET`；具体错误以 Go 实际响应为准，不在 Java 转成成功。

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
