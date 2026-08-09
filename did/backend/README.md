# DAVEX Gov-DID 后端

本目录是 DAVEX DID 新功能的 Go + Gin 实现。它直接复用 `sdk-go/client`、`govdid`、`lsag`、`revocation` 与共享 `protocol`，不启动或包装 `did-terminal` 进程。

## 核心边界

- 每个 actor 使用独立 ChainMaker SDK 配置，链上写交易的 `Origin()` 与其业务 DID 控制者一致；
- 完整 DID 文档、策略 Merkle 语义、完整 VC、认证会话和撤销草案保存在权限为 `0600` 的本地状态文件；
- DID、角色、策略锚、VCProof、群组、密钥映像、委员会和撤销结果以链上状态为最终依据；
- 普通 actor 令牌只能使用自己的 ChainMaker 身份；配置为 `admin: true` 的原型管理员可通过 `X-Actor-Alias` 选择已配置的执行主体，但合约看到的仍是该主体真实 `Origin()`；
- 委员会批准必须切换到不同委员执行主体后分别调用批准接口，`/api/revocation/execute` 不会集中加载委员列表自动代签；
- HTTP 请求、响应和日志均不包含协议私钥或 ChainMaker 私钥。
- 后端初始化 SDK 时关闭其默认 DEBUG 参数日志，避免完整 VC/VP 被上游 SDK 写入 `sdk.log`。

## 配置与启动

```bash
cp configs/backend.example.yml configs/backend.yml
printf '%s' 'replace-with-a-random-token' | shasum -a 256
# 把输出哈希填入 actor.token_sha256，并为各 actor 配置独立 sdk_config_path；
# 原型管理员 actor 可设置 admin: true，普通 actor 保持 false。
# SDK 配置含相对证书路径时，用 sdk_working_dir 指明其解析基准目录。
go run ./cmd/server --config ./configs/backend.yml
```

可用环境变量：`DID_BACKEND_ADDRESS`、`DID_CONTRACT_NAME`、`DID_BACKEND_STATE`。

公开健康检查为 `GET /api/health`。其他接口需要：

```text
Authorization: Bearer <actor token>
```

管理员登录后可在不更换 token 的情况下选择执行主体：

```text
X-Actor-Alias: <configured actor alias>
```

该能力只复用管理员会话，不合并链上身份。DID 注册、VC 签发、匿名认证和委员会批准仍分别由所选 actor 的独立 ChainMaker 证书签名。普通 actor 发送其他别名会返回 403。

核心 API 与 DAVEX `contracts/did-http.md` 保持对应：

- 身份与权限：`generateDID`、`registerDID`、DID 查询、VC 签发、VP 验证、策略证明验证；
- 匿名认证：群组创建/入群/查询、隐私 VP 生成/验证、上下文化密钥映像计算；
- 凭证撤销：事件签发方登记、委员会创建/成员/门限、分步批准、原子撤销和日志查询。

启动后可通过 `http://localhost:8081/swagger/` 打开 Swagger UI，原始契约位于 `/swagger/openapi.yaml`。完整支撑路由见 `internal/httpapi/router.go`。DAVEX 对外路径与响应适配以仓库根目录的 `contracts/did-http.md` 为准。

## 验证

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/server
```
