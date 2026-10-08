# DAVEX DID 系统集成设计与变更边界

> 文档状态：前端页面基线已确认；已有接口待真实链路验证，匿名认证与撤销接入待完成
>
> 适用页面：`DAVEX_ui` 的 `/did` 路由
>
> 更新日期：2026-10-08
>
> 核心原则：DID 是 DAVEX“数据流通”下的一个独立功能，保持 DAVEX 现有导航与视觉风格，不改造其他业务流程。
>
> 合约边界：以参考项目已部署的 `simple-did` 合约为目标，复用 DAVEX 中对应的协议实现；不为简化页面而修改、升级或重新部署合约。

## 1. 文档目的

本文档记录已经确认并完成的 DID 前端结构、各功能的正确业务流程、当前实现状态以及后续 Java/Go 接入边界，供后续编码、联调和验收使用。

本文档中的“合约”如无特别说明，均指 ChainMaker 上的 `did/simple-did` 智能合约；HTTP 请求字段和路径属于“接口契约”，两者不得混淆。

### 1.1 参考项目与 DAVEX 的关系

参考项目位于 `/Users/dengruotao/Desktop/毕业论文/fduthesis/code/did-contract-master`。它是 DAVEX DID 能力的实现依据，不是要求 DAVEX 复制其独立产品形态。后续排查协议行为时，应优先对照参考项目的当前源码及其实际部署配置，不以参考前端的演示结果代替真实链路验证。

| 参考项目 | DAVEX 对应位置 | 集成取舍 |
|---|---|---|
| `simple-did/` | `did/simple-did/` | 复用已部署合约的接口和校验规则；本阶段不改合约、不重新部署。 |
| `sdk-go/`、`protocol/` | `did/sdk-go/`、`did/protocol/` | 复用链客户端、签名、Merkle 策略、DID-LSAG、撤销签名与协议编码；不迁入论文性能实验和独立终端作为运行依赖。 |
| `backend/` | `did/backend/` | 复用 Go HTTP 服务与本地私密状态模型，作为 Java 的 DID 上游；匿名和撤销的 Go 路由已经存在，优先接通而非重写。 |
| `frontend/` | `DAVEX_ui/src/views/did/` | 只参考业务步骤和字段；参考前端当前使用浏览器内 mock 数据，不是链上联调证明，也不搬入其独立导航或视觉样式。 |

参考项目按身份、系统、凭证、联合认证、匿名认证、撤销展示六个模块；DAVEX 将系统初始化和治理留作受控准备操作，将策略并入“身份管理”，对用户只展示第 2 节确定的五个功能。**简化的是入口、表单和步骤，不是链上授权、密码学验证或 actor 隔离。**

## 2. 已确认的产品基线

### 2.1 导航与整体样式

- 保留 DAVEX 的深色侧边栏、蓝色顶栏、白色卡片和 Element Plus 组件风格。
- “数据流通”下只保留一个 `DID身份认证` 入口，对应 `/did`。
- 不增加 DID 一级导航、二级导航或独立侧边栏。
- 不恢复已隐藏的旧“认证管理”导航组。
- 不修改文件传输、比对、安全多方计算、联邦学习等现有页面和路由。

当前导航关系：

```text
数据流通
├─ 文件传输
├─ 比对
├─ 安全多方计算
├─ 联邦学习
└─ DID身份认证  -> /did
```

### 2.2 `/did` 页面结构

页面内部固定为 5 个功能标签：

1. 身份管理。
2. 可验证凭证管理。
3. 身份与权限联合验证。
4. 匿名认证。
5. 凭证撤销。

页面顶部不展示以下内容：

- “DID 身份与凭证系统”标题栏。
- Java、Go 或 ChainMaker 可达状态。
- enabled、ready、端口和配置路径等运行信息。
- 刷新服务状态按钮。

当前操作身份使用右上角 actor 下拉框，不使用自由文本输入。actor 切换只影响 DID 请求中的 `X-DID-Actor`，不改变 DAVEX 登录用户、JWT 或原有权限体系。

### 2.3 页面文案与视觉约束

- 功能标题下不放“创建和查询……”“用于……”等解释性句子。
- 卡片标题只保留操作名称，不在右上角重复放置“创建、查询、签发、验证、撤销、进度”等标签。
- 三步认证流程可以保留数字 `1/2/3`，数字仅表示执行顺序。
- 面向用户使用业务语言，不展示“获取挑战”“获取认证上下文”等密码学或协议术语。
- nonce、签名、Merkle 证明、`qID`、`LHash`、密钥映像和完整 JSON 默认隐藏。
- 操作成功优先展示“凭证有效”“身份与权限验证通过”“匿名资格验证通过”等业务结论。
- 原始返回数据只放在“查看技术详情”折叠区域。
- 未接入的操作不得伪造成功结果。

## 3. 五个功能的页面设计

### 3.1 身份管理

“身份管理”内部使用 `DID 管理` 和 `策略管理` 两个页面内切换项，不增加侧边导航。

#### DID 管理

当前页面包含：

- 生成身份材料。
- 注册 DID。
- 查询 DID。
- 查询角色。
- DID Document 高级设置折叠区。

私钥由 Go backend 本地管理，前端不得生成、保存或返回 P-256、Ristretto255 以及 ChainMaker 私钥。

#### 策略管理

当前页面包含：

- 登记策略。
- 查询策略。
- 停用策略。
- 策略有效期高级设置。

最小业务字段为策略 ID、颁发者 DID、部门角色、授权范围、数据级别和允许动作。Merkle 树、`policyHash` 和成员证明仍由 Go SDK 处理。

### 3.2 可验证凭证管理

当前页面包含：

- 签发 VC。
- 查询 VC。
- 验证 VC。
- 标记 VC 是否允许用于匿名认证。
- 在“匿名资格设置”折叠区中，将当前 VC 加入指定匿名资格群组。

匿名资格群组与 VC 的关系发生在认证准备阶段：

```text
签发 anonymousEligible VC
    -> 将 VC 绑定到资格群组
    -> 持有者后续使用该群组生成匿名凭证展示
```

匿名认证主流程不直接提交或展示具体 VC。加入群组时仍必须由 Go backend 和现有合约验证 VC 状态、策略、签发方、持有者公钥及群组资格。

### 3.3 身份与权限联合验证

用户界面统一使用以下三步名称：

1. 发起联合验证。
2. 生成凭证展示。
3. 验证身份与权限。

对应的底层流程为：

```text
验证方签发 nonce
    -> 持有者选择 VC 并生成 StandardVP
    -> 验证方提交 StandardVP 与访问条件
    -> 合约验证 DID、VC、策略、签名和 nonce
```

交互要求：

- 第一步返回的 nonce 自动传入第二步。
- 第二步返回的标准 VP 自动传入第三步。
- 用户不手工复制或粘贴 VP JSON。
- 不新增绕过不同 actor、独立证书或链上权限的“一键成功”接口。

### 3.4 匿名认证

匿名认证不是“隐藏页面上的 DID 文本”，而是实际生成并验证 `PrivacyVP`。

#### 认证准备

- 匿名资格群组由初始化流程或管理员预先创建。
- VC 必须设置 `anonymousEligible`。
- VC 必须通过“可验证凭证管理”的“匿名资格设置”加入群组。
- 群组创建、启停和成员明细不作为普通认证主流程展示。

#### 用户主流程

界面统一使用以下三步名称：

1. 发起匿名认证。
2. 生成匿名凭证展示。
3. 验证匿名资格。

底层对应关系：

```text
验证方签发 nonce
    -> 持有者选择资格群组
    -> Go backend 根据 holder DID 读取本地 LSAG 私钥
    -> 生成不包含 holder DID 和具体 VC 的 PrivacyVP
    -> 验证方提交 PrivacyVP、qualification 和访问条件
    -> 合约验证群组锚、DID-LSAG、nonce、权限和密钥映像
```

第二步的页面字段为：

- 资格群组 ID。
- 持有者 DID。
- 可选的匿名 VP ID。

持有者 DID 只用于 Go backend 定位本地身份和 LSAG 私钥，不得写入最终 `PrivacyVP`，也不得返回给验证方作为认证结果。

`PrivacyVP` 的核心内容仍由现有实现生成：

- `groupID`。
- `policyID`。
- `qID`。
- `LHash`。
- 包含 nonce、verifier、purpose 和时间片的上下文。
- DID-LSAG 环签名及上下文化密钥映像。

验证结果只展示“匿名资格通过/拒绝”。环签名、`qID`、`LHash` 和密钥映像仅允许出现在技术详情中。

### 3.5 凭证撤销

普通用户主流程：

1. 选择需要撤销的 VC。
2. 选择撤销原因并填写补充信息。
3. 创建撤销申请。
4. 展示批准数量和门限。
5. 委员通过右上角 actor 切换分别批准。
6. 达到门限后执行撤销。
7. 查询 VC 状态和撤销记录。

事件签发方白名单、委员会配置、委员任期、轮换日志、Schnorr 批准签名等内容不放在普通主流程中，但底层校验不得删除或在前端模拟。

## 4. 当前实现状态

### 4.1 已完成

- [x] 保持 DAVEX 原有导航，只保留 `/did` 单一入口。
- [x] 删除 DID 独立标题栏和服务可达状态。
- [x] actor 改为页面右上角下拉选择。
- [x] 页面重组为 5 个业务功能标签。
- [x] 身份管理整合 DID 管理和策略管理。
- [x] 现有 DID、策略、VC 和标准 VP 请求继续使用 `DAVEX_ui/src/api/did.js`。
- [x] 联合验证按“发起联合验证、生成凭证展示、验证身份与权限”展示。
- [x] 匿名认证按“发起匿名认证、生成匿名凭证展示、验证匿名资格”展示。
- [x] 可验证凭证管理中增加“匿名资格设置”入口。
- [x] 删除非必要说明句和卡片右上角重复文字标签。
- [x] 操作结果改为业务摘要加可折叠技术详情。
- [x] 前端生产构建通过。

### 4.2 已有且可继续复用的接口

- DID 生成、注册和查询。
- 策略登记、查询和停用。
- VC 签发、查询和验证。
- nonce 签发。
- 标准 VP 生成和验证。
- actor session、角色查询和更新。

### 4.3 尚未接入 DAVEX DID Java 链路

- 匿名资格群组查询。
- 将 VC 加入匿名资格群组。
- 匿名凭证展示生成。
- 匿名凭证展示验证。
- 密钥映像查询。
- 撤销申请创建和查询。
- 委员批准。
- 执行撤销。
- 撤销记录与最终 VC 状态查询。

当前前端状态：匿名认证第一步复用已有 nonce 接口；匿名资格绑定、匿名凭证展示生成/验证和全部撤销按钮仍是页面流程占位，不会伪造成功响应。

### 4.4 当前联调前提与已知差异

- 参考项目现有 `backend/configs/backend.yml`、各 actor 的 ChainMaker SDK 配置及 `backend/data/backend-state.json`；DAVEX 的 `did/backend/configs/backend.yml`、SDK 实际配置和本地状态尚未准备。示例 YAML 中的 DID、证书路径和 token 哈希不能直接用于真实联调。
- 参考项目当前配置的 `bootstrap`、`issuer`、`holder`、`verifier` 对应现有 `did:govdid:smoke:*` 身份；DAVEX 页面预填的 `did:gov:*`、策略 ID 和 VC ID 只是表单示例，不能假定已存在于链上。
- DAVEX Center 本地配置端口为 `9999`，而 `contracts/did-http.md` 仍写 `9900`；本地联调应以实际运行端口为准，并在扩展接口契约时修正文档。前端缺少本地 `env-config.js` 时，请求默认指向远程 `10.176.37.50:4090`，不会自动连接本地 Center。
- 代码路径、接口和静态构建存在，不等于真实链路已通过。必须分别验证 Go、Java 和页面实际请求，不将参考项目的 mock 前端或构建通过当作功能验收。

## 5. 技术架构与职责边界

```text
DAVEX_ui /did
    -> DAVEX_center DID Controller
    -> DAVEX_base DidService / DidBackendClient
    -> did/backend
    -> did/sdk-go
    -> did/simple-did
    -> ChainMaker
```

### 5.1 前端

前端负责：

- 5 个功能的表单、步骤状态和结果展示。
- actor 选择和 `X-DID-Actor` 请求头。
- 自动传递 nonce、标准 VP、匿名 VP 和撤销草案 ID。
- 隐藏中间密码学对象，将原始数据放入技术详情。

前端不负责私钥管理、签名生成、凭证有效性判断、委员会计数或链上权限判断。

### 5.2 Java

Java 作为薄代理和 DAVEX 统一响应适配层：

- 对前端提供 `/api/v1/did/**`。
- 转发 actor alias、request ID 和 JSON 请求。
- 将 Go backend 响应转换为 `code/message/data`。
- DID 不可用时只返回 DID 专用错误，不影响其他 DAVEX 功能。

Java 不实现 Merkle、P-256、DID-LSAG、Schnorr 或 ChainMaker 合约逻辑。

### 5.3 Go backend 与 SDK

- `did/backend` 负责 actor、本地私密状态、完整策略/VC 数据和多步流程编排。
- `did/sdk-go` 负责 ChainMaker 调用、签名、Merkle 证明、DID-LSAG 和撤销批准签名。
- 优先复用已有服务方法和接口；仅在现有接口不能安全支持页面流程时增加最小编排能力。

### 5.4 智能合约

`did/simple-did` 作为不可变的既有链上能力复用，继续负责：

- DID、角色、策略和 VC 状态。
- nonce 与防重放。
- 匿名群组锚、DID-LSAG 验证和密钥映像消费。
- 撤销委员会、门限批准和原子撤销。

禁止事项：

- 修改合约代码。
- 增加合约方法来适配页面。
- 升级或重新部署合约。
- 删除或弱化现有合约校验。

如果页面流程与现有合约不匹配，应调整页面或在 Java/Go 层进行编排。

### 5.5 不混用旧链路

`/did` 页面应继续使用 `DAVEX_ui/src/api/did.js` 和 `DAVEX_base` DID 链路。除非后续明确制定迁移方案，不得为了快速接通匿名认证而混用旧的 `DAVEX_ui/src/api/goSdk.js`、`/api/v1/control/**` 或另一套 Go backend 业务模型。

### 5.6 复用现有链、合约与本地状态

参考项目的 `simple-did` 是当前链上合约的实现依据；DAVEX 不需要为了页面简化而重新部署。能否复用**当前部署实例**，仍须以实际节点地址、链 ID、合约名、SDK 身份、合约路由和 `/api/ready` 及真实查询结果验证，不能只凭源码相似作出“已连接”的结论。

Go backend 本地状态保存 DID 协议私钥、完整策略及证明、完整 VC 和撤销草案；链上只保存对应锚、角色和最终状态。因此仅复用 SDK 连接配置，不能自动生成现有 DID 的 VP 或管理已有 VC。推荐在不改变参考项目工作区的前提下，为 DAVEX 后端准备**隔离的受限权限状态副本和匹配的 actor/SDK 配置**；配置、token 明文、证书、私钥和状态文件均不得提交仓库。不得让两个 backend 同时写同一状态文件。使用隔离副本做写链测试时采用唯一测试 ID，并核对源项目与 DAVEX 状态可能发生的分叉。

已有 ChainMaker 证书若已控制一个链上 DID，可用于该 DID 的查询和后续授权操作；要演示“生成并注册新的 DID”，须准备未绑定其他 DID 的独立 ChainMaker 身份及相应 actor 配置，不能用已绑定地址重复注册。

DAVEX 前端的 actor 下拉框仅选择执行主体，不构成授权。Go backend 仍以 token、actor 的独立 SDK 证书和链上权限校验。用管理员 token 切换多个 actor 只适合受控本地演示，不应视为 DAVEX 普通登录用户已获得安全的多主体授权。第一轮以 Center 接入为准；只有确认需要 agent 提供 DID 页面时再同步扩展 agent。

## 6. 接口契约改造范围

实施匿名认证与撤销前，必须先更新 `contracts/did-http.md`，再实现 Java 和前端调用。

至少需要定义以下操作的 path、method、actor、请求字段、响应字段和错误码：

- 查询匿名资格群组。
- 将 VC 加入匿名资格群组。
- 生成匿名凭证展示。
- 验证匿名凭证展示。
- 查询密钥映像消费状态。
- 创建和查询撤销申请。
- 委员批准撤销申请。
- 执行撤销。
- 查询撤销记录和 VC 最终状态。

接口原则：

- 优先直接映射 `did/backend` 已有能力。
- nonce、VP、qualification 和草案 ID 由前端自动回填，不要求用户粘贴 JSON。
- 不为了减少按钮数量绕过 actor 隔离。
- 不新增消息队列、工作流引擎、网关或独立密钥服务。

撤销页面当前只填写 VC、事件类型、原因等业务字段，而 Go 创建草案还需要群组、事件签发方及撤销作用域。接入时应从受控预置数据与当前 actor 自动确定这些值，或在必要时放入高级设置；“原因”若需要持久化，应先在 HTTP/本地状态契约中定义，不能假装已写入现有链上合约。批准数量和门限必须读取 Go 状态，不使用固定页面数字。

## 7. 后续计划修改范围

### 7.1 接口契约

- `contracts/did-http.md`

### 7.2 前端

- `DAVEX_ui/src/api/did.js`
- `DAVEX_ui/src/views/did/didWorkspace.vue`
- `DAVEX_ui/src/views/did/components/**`（仅在拆分后能明显降低复杂度时新增）

保持不变：

- `DAVEX_ui/src/views/layout/newLayoutContainer.vue` 的导航结构。
- `DAVEX_ui/src/router/index.js` 的单一 `/did` 路由。

### 7.3 Java

- `DAVEX_base/src/main/java/DavexBase/did/**`
- `DAVEX_center/src/main/java/DavexCenter/module/did/**`
- `DAVEX_agent/src/main/java/DavexAgent/module/did/**`（仅当 agent 确实需要同步暴露时）

### 7.4 Go

- `did/backend/**`（优先复用，必要时补充查询或最小编排）
- `did/protocol/**`（原则上不修改，继续复用现有链上协议模型；新增 HTTP DTO 放在 backend 层）
- `did/sdk-go/**`（原则上不修改）
- `did/simple-did/**`（禁止修改）

## 8. 下一阶段实施顺序

先让已存在的 DID、策略、VC 和标准 VP 链路真实运行，再补匿名与撤销；不要为了测试匿名功能先改动合约或 DAVEX 其他业务。

### 8.1 第一轮：现有接口端到端联调

1. 确认参考项目正在使用的 ChainMaker 节点、链 ID、已部署合约名及每个 actor 的 SDK 身份；为 `did/backend` 准备与链上既有 DID 相匹配的私有配置和隔离的本地状态。操作前备份状态，不直接修改参考项目；不得将密钥或 token 写入 Git。
2. 启动 DAVEX 的 `did/backend`（默认 `8081`）。先检查公开的 `GET /api/health`，再带 actor token 检查 `GET /api/ready`、`GET /api/system/session` 和现有 DID 查询。`health` 只证明 HTTP 存活，`ready` 才检查链和合约；必要时用参考 backend 的同类只读结果定位配置差异。
3. 启动 DAVEX Center，设置 `DID_ENABLED=true`、`DID_BACKEND_URL=http://localhost:8081`、`DID_BACKEND_TOKEN`。先确认 Center 自身所需的数据库等依赖可用，再经实际端口（当前本地配置 `9999`）检查 `/api/v1/did/health` 和 `/api/v1/did/ready`。本阶段不要求启动 `DAVEX_agent`。
4. 将 DAVEX 前端的运行时 API 地址指向该 Center，启动 `/did` 页面；从 session 加载真实 actor 列表，测试时使用实际已存在的 DID/策略/VC ID，不直接采用页面示例值。
5. 按“查询现有 DID → 查询角色/策略 → 查询与验证 VC → 验证方发起联合验证 → 持有者生成标准 VP → 验证方验证”完成第一条闭环。若要测试新 DID 注册，另备未绑定 DID 的 ChainMaker 证书与 actor。

### 8.2 第二轮：匿名认证与撤销

1. 先在 `contracts/did-http.md` 固化匿名资格绑定、PrivacyVP、撤销申请/批准/执行/查询的 Java↔Go path、字段、actor、响应和错误码，同时纠正已发现的 Center 端口差异。
2. 用 `did/backend` 已有接口直接验证匿名最小闭环：受控初始化资格群组、签发允许匿名的 VC、由签发方绑定成员、验证方签发 nonce、持有者生成 `PrivacyVP`、验证方提交资格和访问条件完成链上验证；确认 nonce 与上下文化密钥映像被消费。
3. 直接验证撤销最小闭环：受控预置事件签发方和委员会、创建草案、不同委员 actor 独立批准、达到门限后执行、查询 VC 和群组/撤销记录。不得通过 Java 或前端代替委员签名。
4. 在 `DAVEX_base` 与 Center DID Controller 中补最薄的 HTTP 转发，再扩展 `DAVEX_ui/src/api/did.js`，替换匿名资格绑定、隐私 VP 与撤销页面的占位操作；自动传递 nonce、VP、qualification 和草案 ID，普通用户界面不增加系统管理导航。
5. 验证不同 actor 确实使用不同的 ChainMaker 证书与 DID 密钥、无权限操作被拒绝；停止 DID backend 后检查只有 DID 功能失败，DAVEX 其他页面仍可正常运行。

## 9. 验收标准

### 9.1 页面基线

- [x] “数据流通”下只有一个 DID 入口。
- [x] 页面内部包含 5 个功能标签。
- [x] actor 选择位于右上角。
- [x] 不展示服务可达性和独立 DID 标题栏。
- [x] 不展示非必要说明句和重复操作标签。
- [x] 联合验证和匿名认证使用统一的业务语言。

### 9.2 功能闭环

- [x] DID、策略、VC 和标准 VP 保留现有调用入口。
- [ ] Go `/api/ready` 和 Java `/api/v1/did/ready` 在现有链与合约上实际成功。
- [ ] 经 DAVEX `/did` 页面完成至少一条现有 DID 查询、VC 验证和标准 VP 联合验证；不以接口存在或构建通过代替验收。
- [ ] VC 可以通过 DAVEX DID 链路加入匿名资格群组。
- [ ] 匿名认证实际返回由 Go backend 生成的 `PrivacyVP`。
- [ ] 最终 `PrivacyVP` 和验证结果不包含 holder DID 或具体 VC。
- [ ] 匿名验证会校验并消费 nonce 与上下文化密钥映像。
- [ ] 凭证撤销可以完成申请、独立批准、执行和结果查询。
- [ ] 多 actor 操作使用各自独立的 ChainMaker 证书和 DID 密钥。
- [ ] 测试使用的后端状态和私钥不进入 Git、不与参考 backend 并发共享同一状态文件。

### 9.3 兼容性与安全边界

- [ ] 不修改、升级或重新部署 `did/simple-did`。
- [ ] 不在前端或 Java 保存私钥。
- [ ] 不修改现有 JWT、ABAC 和 center-agent 交互方式。
- [ ] DID backend 不可用时只影响 DID 页面。
- [ ] 文件传输、比对、MPC 和联邦学习页面保持不变。

## 10. 尚待确认的问题

1. actor 下拉列表最终完全由 session 接口返回，还是保留固定演示项作为不可用时的回退。
2. “生成身份材料”和“注册 DID”继续保留两个按钮，还是在接口稳定后合并为一个前端连续操作。
3. 匿名资格群组由部署初始化脚本创建，还是提供仅管理员可见的高级设置入口。
4. 匿名资格群组在页面上使用自由输入、可搜索下拉框，还是根据当前 actor 和 VC 自动推荐。
5. 撤销门限批准在演示时是否要求现场切换多个真实 actor 完成。
6. `DAVEX_agent` 是否需要与 center 同时暴露匿名认证和撤销接口。

## 11. 明确不做的事

- 不修改 DAVEX 全局导航和页面主题。
- 不把参考项目的独立前端布局搬入 DAVEX。
- 不把“系统管理”作为普通用户功能。
- 不向普通用户展示运行端口、配置路径或服务健康详情。
- 不在前端保存 DID、ChainMaker 或 Go backend 私密信息。
- 不在 Java 中重写密码学和智能合约逻辑。
- 不在前端模拟委员会批准数量或伪造链上成功。
- 不强制现有 DAVEX 请求携带 VC 或 VP。
- 不将 DID 强制嵌入文件传输、比对、MPC 或联邦学习流程。
- 不修改 `did/simple-did`。

## 12. 一句话原则

> DAVEX 只保留一个 DID 入口，以 5 个简洁业务功能复用现有 DID 合约能力；页面可以简化，链上合约、actor 隔离、私钥边界和最终验证不得弱化。
