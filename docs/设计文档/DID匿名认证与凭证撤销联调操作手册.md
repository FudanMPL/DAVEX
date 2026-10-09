# DAVEX DID 功能使用手册

本手册说明 DAVEX `/did` 页面五个功能区的用法，并给出可复用的演示数据。实现沿用现有 ChainMaker `simpleDidTest1` 合约；页面通过 DAVEX Center 调用 `did/backend`，不需要修改或重新部署合约。架构和变更边界见 [DID系统集成设计与变更边界](DID系统集成设计与变更边界.md)。

截至本次整理，联合验证和匿名认证已按页面示例完成手动测试。下文的撤销示例是**独立的新数据方案，尚未创建或执行**；不要将其误当成已可直接撤销的现有凭证。

## 使用前

- 本机入口：前端 `http://localhost:5173/did`；Center 当前使用 `http://localhost:9999`；Go DID backend 使用 `http://127.0.0.1:8081`。端口以实际运行配置为准。
- 先确认 `GET http://localhost:9999/api/v1/did/ready` 返回 `code: 0`。Go 的 `/api/health` 只检查 HTTP 存活，`/api/ready` 才检查链与合约。页面右上角选择当前操作身份；演示用的 `bootstrap`、`issuer`、`holder`、`verifier` 对应不同链上 DID。
- 本地演示通过私有配置中的管理员 token 切换 actor。下拉框不是生产环境的登录授权；不要将 token、SDK 私钥或私有状态文件放入 Git、截图或文档。
- 页面初始填入的 `did:gov:*`、`case-read` 等是界面占位示例，不能直接用于当前链。下表才是本机演示使用的 DID：

| 页面操作身份 | DID | 常见操作 |
|---|---|---|
| `bootstrap`（系统初始化方） | `did:govdid:smoke:governance` | 创建和执行撤销申请 |
| `issuer`（凭证签发方） | `did:govdid:smoke:issuer` | 策略、VC、匿名群组成员、委员会、撤销批准 |
| `holder`（凭证持有者） | `did:govdid:smoke:holder` | 生成标准或匿名凭证展示 |
| `verifier`（验证方） | `did:govdid:smoke:verifier` | 发起认证、验证展示、撤销批准 |

以下演示数据在整理时经只读查询处于可用状态；以后重用前仍应查询 `Active` / `Valid`、有效期及群组成员状态。**不要停用示例策略、撤销示例 VC 或移除示例群组成员**，否则后面的例子会失效。

| 用途 | 策略 | VC | 匿名群组 |
|---|---|---|---|
| 联合验证 | `davex-ui-policy-20261008051834223` | `davex-joint-vc-20261008052641` | 不需要；此 VC 的 `anonymousEligible=false` |
| 匿名认证 | `policy-screenshot-20260814-01` | `vc-screenshot-20260814-01` | `group-screenshot-20260814-01` |
| 凭证撤销 | 可复用仍有效的 `davex-ui-policy-20261008051834223` | **另行签发全新 VC** | **另行创建全新群组** |

## 1. 身份管理

### DID 管理

选择「身份管理 → DID 管理」。已有 DID 可直接在「查询 DID」输入 `did:govdid:smoke:holder`，点击「查询身份」查看链上文档；点击「查询角色」查看角色信息。部分演示 DID 没有治理角色，角色查询被拒绝不代表 DID 无效。

「创建 DID」需要先在 Go 私有配置中准备与目标 DID 匹配的 actor、链上交易身份及证书；当前四个演示 actor 已分别绑定自己的 DID，不能只在页面输入任意新 DID 就注册。完成前置配置后，使用**未占用的** DID 标识和匹配的 DID Document 点击「生成身份材料」，再点击「注册 DID」，最后查询确认。生成身份材料会在 Go 的本地私有状态保存签名材料；仅有链上 DID Document 不足以让本机以该身份签名。上述联合验证、匿名认证示例已经有可用 DID，手动演示时无需再创建。

### 策略管理

选择「身份管理 → 策略管理」。可用 `davex-ui-policy-20261008051834223` 点击「查询策略」，确认签发方为 `did:govdid:smoke:issuer`、状态为 `Active`。示例权限为：

| 部门角色 | 授权范围 | 数据级别 | 允许动作 |
|---|---|---|---|
| `community_correction_officer` | `community_correction` | `restricted` | `read`、`approve` |

如果要登记新策略，切换 `issuer`，填写唯一策略 ID、颁发者 DID、权限字段和覆盖当前时间的有效期后点击「登记策略」；以后以返回/查询到的真实策略 ID 为准，不仅凭表单输入判断。「停用策略」会影响引用该策略的示例，请勿对上述两条示例策略使用。

## 2. 可验证凭证管理

选择「可验证凭证管理」，切换 `issuer` 后可以签发 VC。以一张**新 VC** 为例：VC ID 用唯一值，如 `davex-demo-vc-<本轮后缀>`；策略 ID 填仍有效的 `davex-ui-policy-20261008051834223`；颁发者 DID 填 `did:govdid:smoke:issuer`；持有者 DID 填 `did:govdid:smoke:holder`；策略条目索引填 `0`；失效时间填未来的 Unix 秒。若要用于匿名认证，勾选「允许该凭证用于匿名认证」。签发后输入该 VC ID，依次点击「查询 VC」和「验证 VC」，确认链上状态为 `Valid`、验证通过。

只想查看现有数据时，可查询 `davex-joint-vc-20261008052641`；不要重新签发同名 VC。「匿名资格设置 → 将当前 VC 加入群组」只适用于已建好的、与该 VC 策略匹配的群组，并由群组签发方 `issuer` 执行；**签发 VC 不会自动创建匿名群组或完成入群**。

## 3. 身份与权限联合验证

此功能使用**标准 VP**：验证方能看到所出示凭证关联的持有者 DID，不是匿名认证。下面的 VC 是专供本例使用的；不要在撤销示例中使用它。

1. 在「身份与权限联合验证」切到 `verifier`，填验证方 DID `did:govdid:smoke:verifier`、认证用途 `davex-joint-demo`、有效期 `120` 秒，点击「发起联合验证」。认证信息会自动传给下一步。
2. 切到 `holder`，填持有者 DID `did:govdid:smoke:holder`、需要出示的 VC `davex-joint-vc-20261008052641`，点击「生成凭证展示」。
3. 切回 `verifier`，填部门角色 `community_correction_officer`、授权范围 `community_correction`、数据级别 `restricted`、访问动作 `read`，点击「验证身份与权限」。成功时查看结果中的 `verified=true`；不要只看 HTTP 状态。

认证信息有时效，一旦超时请重新从第 1 步发起；不要拿旧 nonce 重复提交。

## 4. 匿名认证

此功能会生成**匿名 VP**，用群组资格证明满足策略，但对外的 VP 不应暴露 `holderDID` 或 `vcID`。下面使用现有一人群组；不要在撤销示例中移除它的成员。

1. 在「匿名认证」顶部填群组 `group-screenshot-20260814-01`。切到 `verifier`，填验证方 DID `did:govdid:smoke:verifier`、认证用途 `davex-anonymous-demo`、有效期 `120` 秒，点击「发起匿名认证」。
2. 切到 `holder`，填持有者 DID `did:govdid:smoke:holder`；「匿名 VP ID」可留空交由系统生成，或每次填新的唯一 ID。点击「生成匿名凭证展示」。这里的持有者 DID 用于 Go 端选取本地私钥，不应出现在对外的匿名 VP 中。
3. 切回 `verifier`，填部门角色 `community_correction_officer`、授权范围 `community_correction`、数据级别 `restricted`、访问动作 `read`，点击「验证匿名资格」。成功结果应有 `verified=true`、`keyImageUsed=true`；技术详情中的匿名 VP 不应包含持有者 DID 或 VC ID。

生成匿名 VP 的前提是：群组 `Active`，VC `Valid` 且 `anonymousEligible=true`，该 VC 已加入群组，策略仍有效。nonce 过期或已被消费时重新发起认证；一次成功验证后，原匿名 VP 不能重放。同一持有者和群组在同一 **5 分钟时间槽**内还会产生相同的 key image，因此马上换 nonce / VP ID 再验证也可能被正确拒绝；若要重复成功演示，等到下一个时间槽，再用新的 nonce 和 VP ID。这样的防重复使用不会撤销 VC。

## 5. 凭证撤销

撤销是不可逆的链上写入。**不要使用**联合验证 VC `davex-joint-vc-20261008052641`，也不要使用匿名认证 VC `vc-screenshot-20260814-01` / 群组 `group-screenshot-20260814-01`。若不打算进行不可逆测试，只需阅读本节，不点击「执行撤销」。

页面负责提交申请、委员批准、执行与查询，也能将 VC 加入已有群组；**新群组和撤销委员会需预先通过 Go backend 创建**。下面 ID 仅为拟用示例，尚未创建；实际操作前先查询是否已占用，并为每一轮换一个唯一后缀。

| 用途 | 独立撤销示例 |
|---|---|
| 待撤销 VC | `davex-manual-revoke-vc-20261009-01` |
| 匿名资格群组 | `davex-manual-revoke-group-20261009-01` |
| 可复用策略 | `davex-ui-policy-20261008051834223`，使用前确认仍 `Active` |
| 已登记的事件类型 | `davex-ui-event-20261008051834223`，使用前确认仍 `Active` |
| 事件签发方 DID | `did:govdid:smoke:governance` |

准备这组新数据：

1. 切换 `issuer`，在「可验证凭证管理」按第 2 节签发**上述新 VC**，勾选匿名资格，确认 `Valid` 且有效期覆盖操作时间。不要修改或覆盖两个既有示例 VC。
2. 通过 Go backend 的 `POST /api/privacy/group/create`，以 `issuer` 创建**上述新群组**，关联同一策略、签发方 DID 和条目索引 `0`；随后可在页面「匿名资格设置」将**这张新 VC**加入**这个新群组**。不要把既有匿名示例 VC 加入该群组。
3. 通过 Go backend 的 `POST /api/revocation/committee/create`，以 `issuer` 为**新群组**建立 `threshold=2` 的委员会，`memberDIDs` 为 `did:govdid:smoke:issuer` 和 `did:govdid:smoke:verifier`；委员任期须覆盖本次操作。Go 的 Swagger 页面位于 `http://127.0.0.1:8081/swagger/`，调用需要本机受控 Bearer token 与相应 `X-Actor-Alias`。页面不提供建群、建委员会入口。

完成准备并核对目标 ID 后，在 DAVEX 页面操作：

1. 切到 `bootstrap`，在「凭证撤销 → 创建撤销申请」填新 VC、新群组、上表已登记的事件类型；「高级设置：事件签发方」填治理 DID。点击「提交撤销申请」，记录自动生成的申请 ID，批准进度应为 `0 / 2`。「撤销原因」不是自由文本，它对应链上登记的 `eventType`。
2. 切到 `issuer`，点击「批准申请」，进度应为 `1 / 2`；再切到 `verifier` 批准，进度应为 `2 / 2`。两次批准必须来自不同的有效委员。
3. 切回 `bootstrap`，再次确认申请指向**新 VC 和新群组**，才点击「执行撤销」。随后点击「查询结果」：新 VC 应为 `Revoked`，申请为 `Executed`，有撤销日志且 `consumed=true`；新群组中的该成员应被移除。
4. 最后分别查询联合验证 VC 与匿名认证 VC/群组，确认前两节演示资源仍 `Valid` / `Active`，匿名群组仍有成员。撤销新 VC 不应使两个既有例子失效。

若执行时页面超时，先用「查询结果」或只读接口核对链上状态，**不要盲目重复点击执行**。若要验证撤销后的匿名资格失效，请重新发起认证、取得新 nonce 并尝试生成新匿名 VP；旧 VP 的重放失败仅能证明防重放，不能单独证明撤销生效。

## 常见问题

- 「服务可访问」但 DID 操作失败：分别检查 Center DID `/ready`、Go `/api/ready`、链 SDK 与合约名；`/health` 成功不足以说明链可用。
- `ACTOR_DID_MISMATCH`：检查右上角 actor 是否与操作方 DID 一致。签发/入群一般用 `issuer`，生成 VP 用 `holder`，验证用 `verifier`，撤销申请/执行用 `bootstrap`。
- 「本地凭证/策略/身份不存在」：Go 不仅依赖链上锚，还依赖其私有状态中的签名与语义材料；不要随意换空状态文件，也不要在参考项目原状态文件上运行 DAVEX。
- 页面刷新后中间数据丢失：认证应重新从发起步骤开始；撤销申请可凭记录的申请 ID 点击「查询结果」恢复进度。

HTTP 字段及错误码以 [DAVEX DID 后端 HTTP 契约](../../contracts/did-http.md) 为准。本手册不记录交易号、截图或逐次实测流水，也不要求为演示修改参考项目、SDK 或链上合约。
