# DAVEX DID 匿名认证与凭证撤销联调操作手册

> 2026-10-08；适用分支 `drt`。本手册以现有 ChainMaker `simpleDidTest1` 合约和 DAVEX 的 `did/backend` 为准，不修改、升级或重新部署合约，也不修改参考项目源码或状态。参考阅读：`did-contract-master/plan/guide/prototype_system_run_and_screenshot_guide.md`；集成边界见同目录的《DID系统集成设计与变更边界》。

## 1. 验收边界

本手册先保留 2026-10-08 的 **Go 后端 ↔ 现有链** 直接验证记录，再记录同日完成的 `DAVEX_ui /did → Center → Go → ChainMaker` 页面实测。两种证据分开列出；参考项目 `frontend` 的 mock 演示不作为链上证明。本实现是受控本地原型，不等同于生产环境的多用户授权和密钥托管方案。

链上写入不可随意回滚，尤其撤销不可逆。仅对本轮新建的策略、VC、群组、事件类型、委员会和草案操作；不复用或撤销参考项目已有 VC。每次运行生成唯一 ID，链上数据保留作审计。勿把 Bearer token、SDK 私钥、完整本地状态或带凭据的命令截图公开。

## 2. 环境准备与隔离

1. 确认当前在 DAVEX `drt` 分支，`git status --short --branch`；保留已有未跟踪的 `DAVEX_ui/public/env-config.js`，不要把它当本轮测试产物删除或提交。
2. Go 服务默认 `127.0.0.1:8081`，DAVEX Center 当前本地运行端口 `9999`，Vite `5173`。Go 需加载四个真实 actor 的 ChainMaker SDK 配置、证书和 `simpleDidTest1` 合约名。不要直接使用仓库内示例 actor/token 值。
3. 参考项目的 `backend/data/backend-state.json` 保存已注册 DID 的本地签名/环签名材料以及策略/VC 语义；先复制到受限权限的隔离副本，不在参考项目原文件上运行 DAVEX。本轮已迁移到 `/Users/dengruotao/Library/Application Support/DAVEX/did/`：目录 `0700`，`backend.yml`、`backend-state.json`、备份及 `audit.jsonl` 均为 `0600`。原 `/tmp` 副本保留作回退；链上写入不能靠恢复文件回滚。配置中的 SDK 路径仍指向参考项目的证书文件，移动参考项目之前需更新这些私有路径。
4. 本地演示使用一个管理员 Bearer token（Go 私有配置保存 SHA-256）；经单独确认后，明文仅存于上述用户私有目录的 `demo-admin-token` 文件（`0600`），不在本文档、Git 或页面保存。它可通过 `X-Actor-Alias` 切换 `bootstrap`、`issuer`、`holder`、`verifier`，四个 actor 仍分别使用原有的链上身份和证书；不新建 DID、不重绑治理角色。此文件一旦泄露便可在该演示后端切换全部 actor，页面下拉框本身不构成授权；仅限本机受控演示。
5. 先用 `/api/health`、`/api/ready`、`/api/system/session`、四个 DID 查询，以及 `issuer`、`verifier` 的角色查询确认服务、链、身份均可用。未配置角色的 `bootstrap`、`holder` 查询角色可能返回合约拒绝，不等同于 DID 无效。请求格式：`Authorization: Bearer $DAVEX_DID_TOKEN`、`X-Actor-Alias: issuer`、`Content-Type: application/json`。下文接口路径均以前缀 `http://127.0.0.1:8081/api` 表示。

| actor | 现有 DID | 本轮职责 |
|---|---|---|
| `bootstrap` | `did:govdid:smoke:governance` | 登记撤销事件签发方、创建/执行草案 |
| `issuer` | `did:govdid:smoke:issuer` | 登记策略、签发 VC、建群/入群、建委员会、批准 |
| `holder` | `did:govdid:smoke:holder` | 生成匿名 VP |
| `verifier` | `did:govdid:smoke:verifier` | 签发 nonce、验证匿名 VP、批准 |

建议设置 `RUN_ID=$(date -u +%Y%m%d%H%M%S)`，为新资源分别使用 `davex-policy-$RUN_ID`、`davex-vc-$RUN_ID`、`davex-group-$RUN_ID`、`davex-revoke-$RUN_ID`、`davex-event-$RUN_ID`。策略 ID 来自登记返回的 `data.anchor.policyID`，不应假设请求中的 `permission.policyID` 就是最终 ID。提交每一步前确认资源尚不存在；交易成功后若响应丢失，先查询状态，不盲目重试写入。

## 3. 匿名认证闭环（先测）

按顺序调用以下 Go 接口，保存每次响应的 `tx.txId`、`requestId` 和必要输出；检查 `code == "OK"`，但最终以链上查询与业务字段为准。

| 步骤 / actor | HTTP 接口和请求重点 | 必查结果 |
|---|---|---|
| 1 策略 / `issuer` | `POST /policy/register`：`issuerDID`、`permissions` 单项，`deptRole=community_correction_officer`、`authScope=community_correction`、`dataLevel=restricted`、`actionSet=["read","approve"]`，时间覆盖当前时刻 | `data.anchor.status=Active`；记下真实 `policyID`；`GET /policy/query?policyID=...` 可读 |
| 2 VC / `issuer` | `POST /vc/issue`：唯一 `vcID`、`holderDID`、`issuerDID`、上一步 `policyID`、`entryIndex=0`、有效 `expiresAt`、`anonymousEligible=true` | `data.proof.status=Valid`；`GET /vc/query?vcID=...` 可读 |
| 3 群组 / `issuer` | `POST /privacy/group/create`：唯一 `groupID`、`policyID`、`issuerDID`、`entryIndex=0` | `GET /privacy/group?groupID=...` 为 `Active`；记录 `memberEpoch`、`lHash` |
| 4 入群 / `issuer` | `POST /privacy/group/member`：`groupID`、`vcID` | 成员绑定为 `Active`，群组公钥集合及锚发生变化 |
| 5 发起认证 / `verifier` | `POST /vp/nonce`：`verifierDID`、`purpose="davex-anonymous-test"`、`ttlSeconds=120` | 记录返回的 `data.nonce`；120 秒内完成后两步；超时只重新获取 nonce |
| 6 生成匿名展示 / `holder` | `POST /privacy/vp/generate`：唯一 `vpID`、`groupID`、`holderDID`、`verifierDID`、`nonce`、同一 `purpose` | 返回 `data.vp` 和 `data.qualification`；VP 对象内不得含 `holderDID` 或 `vcID` |
| 7 验证 / `verifier` | `POST /privacy/vp/verify`：`vp` 原样传回，`qualification` 原样传回，`access` 对应策略权限，例如 `action="read"`、`atTime` 为当前 Unix 秒 | `data.verified=true`、返回 `keyImage`；响应不暴露持有者 DID/VC ID |
| 8 防重放 / `verifier` | `GET /privacy/keyimage?value=...`，随后用**同一** VP 再调用第 7 步 | `used=true`；重复验证应被拒绝，不应再次成功 |

第 1 步权限有效期和第 2 步 VC 到期时间必须覆盖实际测试日；不要照搬参考指南中的固定日期。第 6 步仅匿名 VP 不含持有者身份；Go 输入中的 `holderDID` 用于找本地 LSAG 私钥，不代表链上返回会公开该值。第 7 步应检查返回的 `txID` 和 key image，不能只看 HTTP 200。若验证后 nonce 已消费，**不要复用 nonce** 测新 VP。

## 4. 凭证撤销闭环（只撤销第 2 步新 VC）

| 步骤 / actor | HTTP 接口和请求重点 | 必查结果 |
|---|---|---|
| 1 事件签发方 / `bootstrap` | `POST /revocation/issuer/register`：唯一 `eventType`、`issuerDID=did:govdid:smoke:governance`、`maxAgeSeconds=86400` | `GET /revocation/issuer?eventType=...&issuerDID=...` 为 `Active` |
| 2 委员会 / `issuer` | `POST /revocation/committee/create`：新建 `groupID`、`threshold=2`、`memberDIDs=[issuer DID,verifier DID]`、`termDurationSeconds=86400` | `GET /revocation/committee?groupID=...`：两名有效委员、门限 2 |
| 3 草案 / `bootstrap` | `POST /revocation/requests`：唯一 `draftID`、第 2 步新 `vcID`、同一 `groupID`、`eventIssuerDID`、`eventType`、`scopeType="group"`、`scopeID=groupID` | 状态 `Pending`，目标 VC ID 与本轮记录一致 |
| 4 双人批准 / `issuer`、`verifier` | 各调用一次 `POST /revocation/requests/{draftID}/approvals`（空 JSON 体即可） | 先后批准数 1、2；同一委员不重复批准 |
| 5 执行 / `bootstrap` | `POST /revocation/execute`：`{"draftID":"..."}` | 仅执行一次；记录 `data.credentialHash`、`data.log`、`tx.txId` |
| 6 复核 / 任一 actor | `GET /vc/query?vcID=...`、`GET /privacy/group?groupID=...`、`GET /revocation/logs?vcID=...`、`GET /revocation/consumed?hash=...`、`GET /revocation/requests/{draftID}` | VC `Revoked`、撤销日志存在、credential hash `consumed=true`、草案 `Executed`；群组成员移除且 `memberEpoch` 增加、`lHash` 改变 |

撤销前记下群组 `memberEpoch`、`lHash`，撤销后对比。若执行响应超时，**先做第 6 步只读复核**，确认链上状态后再决定是否处理本地草案，避免对已撤销 VC 二次执行。撤销成功后，旧匿名 VP 的重放仍应失败；如要验证新认证不能由已撤销资格通过，应使用新 nonce，并以生成/验证的实际拒绝为依据，不把单次重放拒绝误判为撤销效果。

## 5. 证据、异常与结果记录

建议截图/留存：健康与链就绪、各 actor session/角色、策略/VC/群组查询、匿名 VP 的脱敏结构、匿名验证成功及 key image 已使用、重放拒绝、批准数到达门限、撤销前后 VC 与群组锚、撤销日志与 consumed 查询。截图隐藏 Bearer token、完整私钥、本地状态 JSON；涉及 `holderDID` 的请求只用于开发证据，匿名对外结果应单独截图。

| 检查点 | 本轮结果 | 证据 / 备注 |
|---|---|---|
| Go 健康、链就绪、四 actor 可用 | 通过 | `/health`、`/ready`、`/system/session` 均为 200；四 DID 均为 `Active`；issuer/verifier 角色符合要求 |
| 新策略、VC、群组、成员上链 | 通过 | `davex-policy-20261008050845026`、`davex-vc-20261008050845026`、`davex-group-20261008050845026`；群组入群后 epoch=1 |
| 匿名 VP 生成与验证 | 通过 | VP 与验证结果均无 holder DID/VC ID；`verified=true`，验证交易 `18dc75ae5216c108ca9243de0816f5725dff0dfef74541f89c4377da0e5e26ce` |
| key image 消费与重放拒绝 | 通过 | `used=true`；同一 VP 重放返回 HTTP 409 `CONTRACT_REJECTED` |
| 事件签发方、2-of-2 委员会、双批准 | 通过 | 本轮新事件类型；issuer/verifier 分别批准，批准数 1→2 |
| VC 撤销、群组更新、日志与 consumed | 通过 | 撤销交易 `18dc75b2412aa0e0caa302c099ad017141be836a8c4544428e2106592a807d26`；VC=`Revoked`，epoch 1→2，公钥数 1→0，`lHash` 改变，日志 1 条，consumed=true，草案=`Executed` |
| 撤销后重新生成匿名 VP | 通过（正确拒绝） | 新 nonce 下返回 HTTP 403 `MEMBER_BINDING_NOT_FOUND`，未生成 VP |
| DAVEX Center / 页面端到端 | 当时未接入，后续已完成 | 见第 6、7 节的新增实现和独立实测 |

这次仅 Go 直测的后缀为 `20261008050845026`；撤销草案 `davex-revoke-20261008050845026`，事件类型 `davex-event-20261008050845026`。其时 DAVEX 可变状态位于 `/tmp/davex-did-run.1ilAek/`，后来完整迁移到第 2 节的用户私有目录。两处均含私密身份材料，勿公开或提交。链上资源不可通过删除任一目录回滚。

失败时先定位层次：`/health` 失败查 Go 进程/端口，`/ready` 失败查 SDK 配置与合约名，`ACTOR_DID_MISMATCH` 查请求 actor，`本地策略/VC/身份不存在` 查状态副本，合约拒绝查角色与资源关联。不要为了让测试通过而修改参考项目状态、跳过签名/门限或改动链上合约。

## 6. DAVEX 接入与日常操作

### 6.1 本轮代码边界

1. 先在 `contracts/did-http.md` 固化 Center `9999`、匿名群组/VP、撤销申请/批准/执行/查询的 Java↔Go 字段、actor 和错误码；未新增或修改链上合约方法。
2. `DAVEX_base` 的 `DidService` 只做原始 JSON 转发，`DAVEX_center` 的 DID Controller 只暴露对应路由；签名、资格、委员门限和链上状态仍由 Go/合约决定。没有把参考项目旧 `/api/v1/control/**` 路由接回 `/did`，也没有扩大 `DAVEX_agent` 范围。
3. `DAVEX_ui/src/api/did.js` 增加以上调用；`didWorkspace.vue` 将入群、匿名 VP、撤销申请、委员批准、执行和查询的占位按钮替换为真实请求。nonce、VP、qualification、草案 ID 自动传递；批准数和门限从真实草案与委员会读取。页面仍只有原有五个功能标签、右上角 actor 下拉框。
4. 现有合约没有“补充说明”字段，本轮移除原先未持久化的页面文本框；撤销原因改为输入已在链上登记的 `eventType`，避免把未保存内容显示为已提交。

### 6.2 服务启动与就绪

保持第 2 节私有文件的权限和路径。下列命令在 DAVEX 仓库根目录运行；`DID_BACKEND_TOKEN` 从受限权限文件读取，`JWT_SECRET` 仍应由本地安全渠道注入，不把明文写入命令历史、文档或 Git。Center 沿用项目现有 `application.yml` 的数据库配置，并从 `application-template.yml` 补充本地模板字段；本轮未改数据库配置文件。不要同时让参考 backend 与 DAVEX Go 写同一状态文件。

```bash
cd did/backend
GOROOT=/usr/local/go /usr/local/go/bin/go run ./cmd/server --config '/Users/dengruotao/Library/Application Support/DAVEX/did/backend.yml'
```

另一个终端运行：

```bash
sh '/Applications/IntelliJ IDEA.app/Contents/plugins/maven/lib/maven3/bin/mvn' \
  -pl DAVEX_center -am -DskipTests package
export DID_BACKEND_TOKEN="$(tr -d '\n' < '/Users/dengruotao/Library/Application Support/DAVEX/did/demo-admin-token')"
# JWT_SECRET 另由当前 shell 的安全环境提供；后续重启应使用独立值
DID_ENABLED=true DID_BACKEND_URL=http://127.0.0.1:8081 \
  java -jar DAVEX_center/target/davex-center-0.0.1.jar \
  --spring.config.location=classpath:/application-template.yml,classpath:/application.yml
```

前端在 `DAVEX_ui` 执行 `npm run dev -- --host 127.0.0.1 --port 5173`；本机未跟踪的 `public/env-config.js` 将 API 地址指向 `http://localhost:9999`，不可误指远端默认 API。检查 `http://127.0.0.1:8081/api/health`、带 Bearer 的 Go `/api/ready`、Center `/api/v1/did/ready`、`/api/v1/did/session` 和 `http://localhost:5173/did`。`health` 只说明 HTTP 存活，`ready` 才说明链/合约可用。

### 6.3 受控准备与页面演示

每轮用新的唯一后缀。使用 `issuer` 在 Go 端登记一条当前有效的策略、签发 `anonymousEligible=true` 的新 VC、按该策略创建新匿名群组；用 `bootstrap` 登记事件签发方；用 `issuer` 为该群组创建包含 `issuer`、`verifier` 的 2-of-2 委员会。具体 Go 请求字段与查询检查见第 3、4 节；这些是部署/演示准备，不新增普通用户导航。已登记的事件类型可在新的测试群组中复用，但只撤销本轮新 VC。

在 `/did` 页面依次操作：

1. 右上角切到 `issuer`；“可验证凭证管理 → 匿名资格设置”输入本轮 VC ID 和群组 ID，点击“将当前 VC 加入群组”，应显示成功。签发方以外 actor 入群应被拒绝。
2. 切到 `verifier`；“匿名认证 → 发起匿名认证”填写实际 verifier DID 和用途，点击发起。nonce 会自动传入第二步，120 秒过期时重新发起。
3. 切到 `holder`；填写实际 holder DID，点击“生成匿名凭证展示”。页面只显示业务状态；技术详情中的 `vp` 不得含 holder DID 或 VC ID。
4. 切回 `verifier`；按策略填部门角色、授权范围、数据级别和动作，点击“验证匿名资格”。结果应为 `verified=true`、`keyImageUsed=true`；同一 VP 在 Center 接口重放应返回 `41005` 且 `upstreamCode=CONTRACT_REJECTED`。
5. 切到 `bootstrap`；“凭证撤销”中填本轮 VC、群组、**已登记**的事件类型。高级设置中的事件签发方 DID 应是治理 DID。提交申请后自动回填草案 ID，显示真实门限 `0 / 2`。
6. 切到 `issuer` 批准一次，再切到 `verifier` 批准一次；页面显示 `1 / 2 → 2 / 2`。不要用一个 actor 代替两名委员。
7. 切回 `bootstrap`，确认只针对本轮 VC 后执行不可逆撤销。页面自动查询草案、委员会、VC、日志和 consumed；复查 VC=`Revoked`、草案=`Executed`、日志存在、consumed=true、群组成员消失且 epoch 增长。

遇到超时，先使用“查询结果”和只读链上接口确认状态，不重复点击执行。页面表单状态不持久化；刷新后可用本手册记录的草案 ID 重新查询。页面预填的 `did:gov:*`/`case-read` 只是布局示例，务必替换为实际 `did:govdid:smoke:*` 和本轮新 ID；真实按钮会写入链，绝不能用既有生产 VC 试点。当前管理员 token + actor 下拉仅限本机受控演示；若要开放给真实普通用户，必须先设计独立登录身份与 actor 授权绑定，不得复用演示 token。

## 7. 本轮 DAVEX 完整链路实测记录（2026-10-08）

| 检查项 | 结果 | 可复核证据 |
|---|---|---|
| 构建与启动 | 通过 | Vite `npm run build`、Center Maven package 成功；Go 8081、Center 9999、Vite 5173 均可用；Center `/ready` 返回 `code=0` |
| 原有 DID / VC / 标准 VP 回归 | 通过 | 新 VC `davex-joint-vc-20261008052641` 保持 `Valid`；经 Center 查询 holder DID 为 `Active`、验证 VC 成功、verifier 发 nonce、holder 生成标准 VP、verifier 验证返回 `verified=true`；交易 `18dc76a4cbbadcf0ca62814a34f28992b5114db22102437d9b8643cc19004d29`。标准 VP 按设计包含 holder DID，不与匿名 VP 混淆 |
| Java→Go→链匿名流程 | 通过 | 新资源 `davex-ui-vc-20261008051834223` / `davex-ui-group-20261008051834223`；Center 入群、nonce、PrivacyVP 生成/验证均返回 `code=0`；VP 不含 holder DID/VC ID，`keyImage.used=true`；重放返回 `41005/CONTRACT_REJECTED` |
| Java→Go→链撤销流程 | 通过 | 草案 `revoke-f40a9747988d9008f7639314`，两个 actor 批准数 1→2；撤销交易 `18dc763a37fad548cad971ca6e051365319efaea8d514ce2948a301c1a6f3d65`；VC=`Revoked`、epoch 1→2、公钥数 1→0、日志 1 条、consumed=true；新 nonce 下匿名 VP 生成被拒绝 |
| `/did` 页面匿名流程 | 通过 | 新 VC `davex-page-vc-20261008052134285`、群组 `davex-page-group-20261008052134285`；页面入群、切换 verifier/holder/verifier、生成 PrivacyVP、验证成功；技术详情显示 `verified=true`、`keyImageUsed=true` |
| `/did` 页面撤销流程 | 通过 | 草案 `revoke-bec254cff892d9428a339cb6`；页面显示 `0/2→1/2→2/2`，切回 bootstrap 执行并自动查询；链上交易 `18dc7680297efb70ca72725224882f7425bc1d6022684dcf893d03b285aa30f1`；复核 VC=`Revoked`、epoch=2、群组成员数 0、日志 1 条、consumed=true；新 nonce 下该群组匿名 VP 生成返回 `41005/MEMBER_BINDING_NOT_FOUND` |
| 重启与故障隔离 | 通过 | Go 状态迁到用户私有目录后重启，Center `/ready` 成功且已撤销 VC 的本地完整凭证仍可读；Go 暂停时 DID `/ready` 返回 `41004`，非 DID `/api/v1/health` 仍返回 `code=0`；恢复 Go 后 `/ready` 再次成功 |
| actor 越权 | 正确拒绝 | 使用 `holder` 尝试入群，Center 返回 `41005/ACTOR_DID_MISMATCH`，无链上写入 |

本轮未修改参考项目、`did/simple-did`、`did/sdk-go` 或 Go backend 代码；现有 Go 接口已足够完成这次集成。运行状态、私有配置、演示 token 及备份均未纳入 Git。当前 `/did` 是受控演示闭环；生产级登录授权、密钥管理、服务自启动和其他 DAVEX 业务的深入回归不在本轮结论内。
