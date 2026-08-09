# DAVEX DID Go SDK 核心库

本目录保留 Go DID backend 运行所需的最小 SDK 源码：

- `client/`：ChainMaker 连接、合约调用和响应归一化。
- `govdid/`：DID 密钥、策略 Merkle 树、VC/VP 签名与本地工作区模型。
- `lsag/`：Ristretto255 DID-LSAG 签名与本地验签。
- `revocation/`：撤销凭证与委员批准签名。

本目录是库，不是独立网络服务。Java 通过 `../backend` 提供的 HTTP API 使用这些能力。

出于模块隔离和凭据安全考虑，仓库不包含实际 ChainMaker 私钥或 crypto-config。请从 `configs/sdk_config.example.yml` 生成本地且被 Git 忽略的 `configs/sdk_config.yml`。
