# 服务器环境配置与 DAVEX 运行基线

## 1. 文档用途

本文档记录 DAVEX 联调服务器的环境、Docker 状态、数据库基线和后续启动约定。当前业务测试拓扑为一个检察院 Center、一个法院 Agent 和一个金融监管机构 Agent，前端在本地运行。

记录日期：2026-09-08（Asia/Shanghai）。

## 2. 服务器信息

| 配置项 | 当前值 |
|---|---|
| 服务器地址 | `10.176.37.50` |
| 主机名 | `dslb50` |
| 操作系统 | Ubuntu 22.04.5 LTS |
| 登录用户 | `drt` |
| SSH 端口 | `22` |
| SSH 认证 | 服务器仅允许公钥认证 |
| 本机 SSH 密钥配置 | `~/.ssh/rsa172` |
| Docker 权限 | `drt` 不在 Docker 用户组，Docker 命令需要 `sudo` |

连接命令：

```bash
ssh drt@10.176.37.50
```

安全约定：SSH、sudo 和 MySQL 密码不写入仓库文档，使用已单独提供的测试环境凭据。

## 3. 服务器运行环境

| 软件 | 版本或状态 |
|---|---|
| Java | OpenJDK `17.0.20` |
| Git | `2.34.1` |
| Maven | 未安装全局 `mvn` |
| Node.js | `12.22.9` |
| npm | `8.5.1` |
| Docker Server | `27.5.1` |
| MySQL | `8.0.46`，systemd 服务 |

本次前端计划在本地启动，因此服务器的旧 Node.js 版本暂不影响联调。

## 4. Docker 当前状态

### 4.1 已停止且保留的 DAVEX 容器

2026-09-08 已执行 `docker stop`，未执行 `docker rm`。

| 容器 | 容器 ID | 停止前端口 | 当前状态 | 重启策略 |
|---|---|---|---|---|
| `davex-agent-backend` | `ec83d90e0cf7` | `4099` | Exited | `no` |
| `davex-agent-frontend` | `43f227c86126` | `4094 → 80` | Exited | `no` |
| `davex-garnet` | `0ba4fbd4edac` | `6000-6002`、`6099-6100` | Exited | `no` |

如需恢复旧容器：

```bash
sudo docker start davex-agent-backend davex-agent-frontend davex-garnet
```

不要使用 `docker rm`、`docker compose down -v` 或删除对应挂载目录。

### 4.2 保持运行的基础设施容器

以下容器未停止：

- 4 个 ChainMaker VM 容器：`VM-GO-didTest-TestCMorg*-cmtestnode*`。
- ChainMaker 管理页面：宿主机端口 `8000`。
- ChainMaker 管理后端：宿主机端口 `9000`。
- ChainMaker 管理数据库容器。
- `mysql-container`：Docker MySQL，宿主机端口为 `3307`。

`mysql-container` 不是本次 DAVEX Java 服务计划使用的数据库。

## 5. MySQL 配置

### 5.1 目标 MySQL 服务

| 配置项 | 当前值 |
|---|---|
| 地址 | `10.176.37.50` |
| 端口 | `3306` |
| 部署方式 | 宿主机 systemd 服务，不是 Docker |
| 版本 | MySQL `8.0.46` |
| 用户 | `drt` |
| 密码 | 使用单独提供的测试环境凭据，不写入仓库 |
| 字符集 | `utf8mb4` |
| 排序规则 | `utf8mb4_0900_ai_ci` |

连接示例：

```bash
mysql --protocol=TCP -h 10.176.37.50 -P 3306 -u drt -p
```

### 5.2 本次测试使用的数据库

只使用带 `drt_` 前缀的数据库：

| 服务 | 数据库 | 表数量 | 当前数据状态 |
|---|---|---:|---|
| 法院 Agent | `drt_agent_court` | 15 | 10 个目录、76 份法院业务文件 |
| 金融监管机构 Agent | `drt_agent_finance` | 15 | 2 个目录、7 份金融监管业务文件 |
| 检察院 Center | `drt_center` | 34 | 已注册两个 Agent，并同步 12 个目录、83 份文件 |

JDBC 地址：

```text
jdbc:mysql://10.176.37.50:3306/drt_agent_court
jdbc:mysql://10.176.37.50:3306/drt_agent_finance
jdbc:mysql://10.176.37.50:3306/drt_center
```

旧的 `drt_agent` 以及无前缀的 `agent`、`center` 数据库仍然存在且未被当前三个服务使用，后续测试不得误用。

### 5.3 数据库重建来源

重建使用仓库中的：

- `SQL/agent.sql`
- `SQL/center.sql`

原始 SQL 会写死以下数据库：

```sql
CREATE DATABASE IF NOT EXISTS `agent`;
USE `agent`;
```

```sql
CREATE DATABASE IF NOT EXISTS `center`;
USE `center`;
```

因此服务器使用适配后的临时副本：

- `/home/drt/davex-db-rebuild/agent_drt.sql`
- `/home/drt/davex-db-rebuild/center_drt.sql`

临时副本在首次重建时包含以下必要修正：

1. 将目标数据库名替换为 `drt_agent`、`drt_center`。
2. `agent.sql` 原第 242 行查询语句缺少分号，重建临时副本已补充分号。
3. `center.sql` 原第 308 行存在未注释的分隔符，重建临时副本已改为 SQL 注释。

上述两处原始 SQL 语法问题已于 2026-09-08 同步修复到仓库，后续不再需要重复修正。

## 6. 数据库备份与恢复

数据库清空前已完成完整备份：

```text
/home/drt/davex-db-backups/drt_agent_drt_center_before_reset_20260908T023157Z.sql
```

备份信息：

| 配置项 | 值 |
|---|---|
| 大小 | 约 71 KB |
| 包含数据库 | `drt_agent`、`drt_center` |
| 表数量 | 46 |
| 数据 INSERT 组数 | 28 |
| SHA-256 | `457848b1116571f0daca05e6d036f198b4306d8bd370cca071630739f82dfe60` |

需要恢复旧数据时，应先停止新 Agent 和 Center，然后执行：

```bash
mysql --protocol=TCP -h 127.0.0.1 -P 3306 -u drt -p \
  < /home/drt/davex-db-backups/drt_agent_drt_center_before_reset_20260908T023157Z.sql
```

恢复会覆盖当前 `drt_agent`、`drt_center`，执行前仍需再次备份和确认。

在写入本次文件传输测试数据前，又创建了一份空库启动基线备份：

```text
/home/drt/davex-db-backups/drt_agent_drt_center_before_file_test_20260908T031000Z.sql
```

其 SHA-256 为：

```text
80c1ab0c249014abd5fdfbd8cd7f88cb3c14254fdee4be4e591ef313cc5cf670
```

切换为法院、金融监管机构双 Agent 拓扑前，另行备份了数据库、配置、数据目录和 PID 文件：

```text
/home/drt/davex-node-migration-backups/20260908T064126Z
```

本次切换中的失败尝试数据和旧联调结果也移动到该目录保存，没有直接删除。

## 7. 服务器上的现有 DAVEX 文件

`drt` 用户目录下存在旧构建产物：

```text
/disk/drt/DAVEX/davex-agent-0.0.1.jar
/disk/drt/DAVEX/davex-center-0.0.1.jar
```

这些 JAR 的构建时间为 2024-12-30，不能视为当前 `0907` 分支产物。后续启动前应在本地按当前分支重新构建并上传，不建议直接复用旧 JAR。

服务器未安装 Maven，因此推荐在本地构建，在服务器只使用 Java 17 运行 JAR。

2026-09-08 已从本地 `0907` 分支重新构建并部署到独立目录：

```text
/disk/drt/DAVEX-0907/bin/davex-agent-0.0.1.jar
/disk/drt/DAVEX-0907/bin/davex-center-0.0.1.jar
```

| JAR | SHA-256 |
|---|---|
| Agent | `254850f2f45b45231375b2d15249f79f4ae57f07b576bbe38dd843ca43b871bc` |
| Center | `320c7d46a1d5c4e5bca27820b676e22f748792098032f5654ea4f7435e27b042` |

旧目录 `/disk/drt/DAVEX/` 中的 2024 年 JAR 未被覆盖。

2026-09-08 为支持本地前端跨域下载，Center 下载接口改为统一返回 `ResponseEntity`：成功时返回文件资源，失败时返回原有 JSON `Body`。同时不再调用 `response.reset()`，避免清除 Spring 写入的 CORS 响应头，也避免文件流写完后再次序列化 JSON。部署前后的 Center JAR 备份为：

```text
/disk/drt/DAVEX-0907/bin/davex-center-0.0.1.jar.backup-20260908T060754Z
/disk/drt/DAVEX-0907/bin/davex-center-0.0.1.jar.backup-20260908T061525Z
```

## 8. 已停止的旧非 Docker Center

服务器此前有一套不属于上述 Docker 容器的旧 Center：

| 配置项 | 当前值 |
|---|---|
| PID | `2578419` |
| 用户 | `root` |
| 工作目录 | `/disk/zkx/DAVEX` |
| 监听端口 | `4090` |
| 启动方式 | `java -Djava.io.tmpdir=/disk/zkx/tmp -jar davex-center-0.0.1.jar` |
| 当前状态 | 已于 2026-09-08 使用 `SIGTERM` 正常停止 |

旧进程及其父进程均已退出，`4090` 已释放；工作目录和 JAR 文件均未删除。

本次 `drt` 测试 Center 后续使用已释放的 `4090`。

## 9. 后续 DAVEX 启动约定

### 9.1 端口

| 服务 | 运行位置 | 当前端口 |
|---|---|---:|
| 法院 Agent | `10.176.37.50` | `8080` |
| 金融监管机构 Agent | `10.176.37.50` | `8083` |
| 检察院 Center | `10.176.37.50` | `4090` |
| Go Backend（如启用） | `10.176.37.50` | `8081` |
| Center 前端 | 本地开发机 | `8082` |

### 9.2 实际部署配置

服务器采用外置 YAML 覆盖仓库内默认配置，不修改 JAR，也不把服务器密码写入 Git：

```text
/disk/drt/DAVEX-0907/config/agents/court/application.yml
/disk/drt/DAVEX-0907/config/agents/finance/application.yml
/disk/drt/DAVEX-0907/config/center/application.yml
/disk/drt/DAVEX-0907/config/davex.env
```

上述文件权限均为 `600`。数据库密码由 `davex.env` 注入 YAML 中的环境变量占位符。

主要覆盖项：

| 配置 | 法院 Agent | 金融监管机构 Agent | 检察院 Center |
|---|---|---|---|
| 服务端口 | `8080` | `8083` | `4090` |
| 数据库 | `drt_agent_court` | `drt_agent_finance` | `drt_center` |
| 服务 ID | `DAVEX-CXX50-GXX5` | `DAVEX-CXX50-GXX6` | `DAVEX-CXX50` |
| 对外 IP | `10.176.37.50` | `10.176.37.50` | `10.176.37.50` |
| 数据根目录 | `/disk/drt/DAVEX-0907/data/agents/court` | `/disk/drt/DAVEX-0907/data/agents/finance` | `/disk/drt/DAVEX-0907/data/center` |
| ChainMaker | 关闭 | 关闭 | 关闭 |
| Garnet | 注册 Bean，不执行任务 | 注册 Bean，不执行任务 | 关闭 |

Agent 现有代码中的 `MpcTaskService` 会强制注入 `GarnetService`，因此把 `garnet.enabled` 设为 `false` 会导致 Agent 无法启动。本次配置保持该 Bean 注册，但没有启动已停止的 Garnet Docker 容器，也不调用 Garnet 任务。启动日志会记录一次 Docker socket 权限/容器不存在异常，该异常已被代码捕获，不影响文件传输。

### 9.3 启动与停止

Center 启动方式：

```bash
cd /disk/drt/DAVEX-0907
set -a
. config/davex.env
set +a
nohup java -jar bin/davex-center-0.0.1.jar \
  --spring.config.additional-location=file:/disk/drt/DAVEX-0907/config/center/ \
  > logs/center.log 2>&1 < /dev/null &
```

法院 Agent 启动方式：

```bash
cd /disk/drt/DAVEX-0907/nodes/court
set -a
. /disk/drt/DAVEX-0907/config/davex.env
set +a
nohup java -jar /disk/drt/DAVEX-0907/bin/davex-agent-0.0.1.jar \
  --spring.config.additional-location=file:/disk/drt/DAVEX-0907/config/agents/court/ \
  > /disk/drt/DAVEX-0907/logs/agent-court.log 2>&1 < /dev/null &
```

金融监管机构 Agent 启动方式与法院 Agent 相同，配置目录改为 `config/agents/finance/`，工作目录改为 `nodes/finance/`，日志使用 `logs/agent-finance.log`。

当前 PID 文件：

```text
/disk/drt/DAVEX-0907/run/center.pid
/disk/drt/DAVEX-0907/run/agent-court.pid
/disk/drt/DAVEX-0907/run/agent-finance.pid
```

停止时先读取 PID 并确认命令行属于本次部署，再发送 `SIGTERM`，不要直接按端口批量杀进程。

### 9.4 本地前端联调地址

本地 Center 前端后续应访问：

```text
http://10.176.37.50:4090/
```

当前前端源码默认回退地址是 `http://10.176.37.50:4090/`，与运行中的测试 Center 一致。

## 10. 文件传输联调基线

测试应用 ID 为 `DAVEX-C1-A1`。法院数据保留行政执法、刑事、民事等多层目录；数据量较大的叶子目录按文件名排序取 15 份，源目录不足 15 份时全部导入。金融监管源数据只有 7 份，因此全部导入。`.DS_Store` 和 `._*` 文件未导入。

已分别验证以下完整链路：

- 法院 Agent 的行政处罚决定书传输到 Center。
- 金融监管机构 Agent 的银行业金融机构法人名单 CSV 传输到 Center。

两条记录均已进入 Center 结果管理区，结果记录 ID 为自增值，不应在前端写死。

## 11. 当前环境结论

- 旧 DAVEX Docker 容器已停止但完整保留。
- ChainMaker、管理后台及 MySQL 基础设施保持运行。
- `drt_agent_court`、`drt_agent_finance`、`drt_center` 已按仓库 SQL 重建并写入业务测试数据。
- 数据库重建前的完整备份已保留。
- 服务器旧 Center 已停止；本次 `0907` 检察院 Center、法院 Agent、金融监管机构 Agent 分别运行在 `4090`、`8080`、`8083`。
- 两个 Agent 到 Center 的文件传输链路均已实际验证成功。
