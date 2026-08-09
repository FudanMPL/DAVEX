# simple-did 主系统合约

本目录实现论文第四章定义的三个主机制：Gov-DID 身份与权限联合认证、DID-LSAG 匿名认证，以及“合规—治理—执行”三层凭证撤销。对比实验路由不属于本合约。

## 模块

- `main.go`：ChainMaker 入口与路由分发；
- `service_identity.go`：DID、系统角色、不可变策略、VC/标准 VP；
- `service_group.go`：`qID` 资格群组、成员绑定、DID-LSAG 验签与密钥映像消费；
- `service_revocation.go`：事件签发方白名单、撤销委员会、门限批准与原子撤销；
- `crypto.go`：P-256 ECDSA、Ristretto255 DID-LSAG 与 Ristretto255 Schnorr 验签；
- `store.go`：链上状态读写与索引；
- `../protocol`：主合约与后续 Go SDK 共用的 DTO 和确定性编码。

## 密码套件

- DID、策略、VC、VP 和撤销凭证单签：ECDSA P-256，ASN.1 DER 签名十六进制编码；
- DID-LSAG：Ristretto255 质数阶群。Ristretto255 建立在 Edwards25519 上，提供规范点编码和无余因子质数阶抽象；算法标识为 `DID-LSAG-Ristretto255-v1`；
- 撤销委员会批准：Ristretto255 Schnorr 独立签名集合，与 DID-LSAG 沿用同一质数阶群；链上逐签名验证并统计互异成员。

Ristretto255 的 `FromUniformBytes` 用于把链接域安全映射到群点，不采用公开标量乘固定生成元的旧实现。

## 调用约定

所有路由统一接收一个名为 `request` 的 JSON 参数，响应为 JSON。主要路由如下。

| 分类 | 路由 |
|---|---|
| DID 与治理 | `RegisterDID`、`UpdateDID`、`DeactivateDID`、`GetDIDRegistry`、`ValidateDID`、`GetDIDByAddress`、`SetGovernanceDID`、`UpdateRoles`、`GetRoles` |
| 策略 | `RegisterPolicy`、`DeactivatePolicy`、`GetPolicy`、`VerifyPolicyMembership` |
| VC/VP | `IssueVC`、`VerifyVC`、`GetVCProof`、`IssueNonce`、`VerifyVP`、`VerifyVPWithPolicy` |
| 匿名群组 | `CreateCredentialGroup`、`AddMemberToGroup`、`GetCredentialGroup`、`GetGroupPublicKeys`、`GetMemberBinding`、`SuspendGroup`、`ActivateGroup` |
| 匿名认证 | `VerifyPrivacyVP`、`CheckKeyImage` |
| 撤销合规 | `RegisterEventIssuer`、`RemoveEventIssuer`、`GetEventIssuer` |
| 撤销治理 | `CreateRevocationCommittee`、`AddCommitteeMember`、`RemoveCommitteeMember`、`UpdateCommitteeThreshold`、`GetRevocationCommittee`、`GetCommitteeRotationLogs` |
| 撤销执行 | `ExecuteRevocation`、`IsCredentialConsumed`、`GetRevocationLogs` |

## 初始化与治理

合约初始化时记录部署交易的 `Origin` 为不可变 bootstrap address。部署者首先注册由该地址控制的 DID，再调用 `SetGovernanceDID` 完成一次性治理 DID 绑定。治理 DID 可通过 `UpdateRoles` 授予：

- `policy_manager`：发布和停用不可变策略；
- `credential_issuer`：签发 VC、建立群组和管理成员；
- `verifier`：签发 nonce 并提交标准或匿名认证结果验证。

普通 DID 不能自行声明机构角色。

## 与旧合约的关键差异

- 同一 `policyID` 不再允许 `UpdatePolicy`，只能停用旧策略并发布新 ID；
- VC 签发必须验证权限中全部动作叶的 Merkle 证明；
- 标准 VP 必须绑定验证者签发且未消费的 nonce、完整 VC、访问请求和权限证明；
- 匿名群组保存 `qID`、`memberEpoch`、`LHash` 和 `(groupID,vcID,holderDID,PK)` 绑定；
- `VerifyPrivacyVP` 执行完整 `LP/RP` 环形挑战链验签，并在同一交易消费 `I_ctx`；
- 不提供公开 `RecordKeyImageUsage`；
- 匿名资格不提供直接 `RevokeVC` 或 `RemoveMemberFromGroup`，只能由 `ExecuteRevocation` 在门限批准后原子撤销；
- 撤销凭证包含 `targetVCID` 和带类型作用域，且执行时验证法定签发方签名、事件窗口、四元组绑定、委员 DID—公钥—任期与批准门限。

## 构建与检查

```bash
make check-simple
```

根目录 Makefile 会忽略 shell 中失效的 `GOROOT` 覆盖，依次执行格式化、单元/集成测试、`go vet` 和 Linux amd64 合约构建。部署前仍应使用与目标 ChainMaker 节点一致的 Go/ABI 环境重新构建。构建产物 `simpleDid` 不进入 Git。
