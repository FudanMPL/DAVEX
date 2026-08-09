# DAVEX DID 子系统

本目录是 DAVEX 的独立 DID 新功能，源代码来自本地 `did-contract-master` 仓库的当前工作区版本。

## 目录

- `backend/`：Go + Gin HTTP 服务，由 DAVEX Java 调用。
- `sdk-go/`：被 backend 进程内引用的 ChainMaker/DID SDK 核心库。
- `protocol/`：SDK 与合约共享的 DTO 和确定性编码。
- `simple-did/`：部署到 ChainMaker 的智能合约。

本次只迁入运行后端和合约需要的最小源码集，不包含原仓库的前端、benchmark、比较实验、二进制、日志、数据和密钥。

## 配置

```bash
cd did
cp sdk-go/configs/sdk_config.example.yml sdk-go/configs/sdk_config.yml
cp backend/configs/backend.example.yml backend/configs/backend.yml
```

1. 修改 `sdk-go/configs/sdk_config.yml`，填写本机 ChainMaker 节点、证书和私钥路径。
2. 为每个 actor 准备独立 SDK 配置和身份。
3. 生成 backend token SHA-256，将哈希写入 `backend.yml` 的 `token_sha256`。
4. Java 只通过 `DID_BACKEND_TOKEN` 获得 token 明文；明文不写入 Git。

## 启动

```bash
cd did/backend
go run ./cmd/server --config ./configs/backend.yml
```

DAVEX 集成默认使用 `http://localhost:8081`。健康检查：

```bash
curl http://localhost:8081/api/health
```

Java 侧默认关闭 DID 功能。联调时显式设置：

```bash
export DID_ENABLED=true
export DID_BACKEND_URL=http://localhost:8081
export DID_BACKEND_TOKEN='<token>'
```

## 验证

```bash
cd did
go test ./protocol/... ./sdk-go/... ./backend/... ./simple-did/...
go vet ./protocol/... ./sdk-go/... ./backend/... ./simple-did/...
go build -o ./backend/did-backend ./backend/cmd/server
```

HTTP 契约见 `../contracts/did-http.md`，架构与变更边界见 `../docs/设计文档/DID系统集成设计与变更边界.md`。
