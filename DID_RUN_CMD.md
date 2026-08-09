# DAVEX DID 启停命令

本地联调需要启动三个进程：Go DID backend、DAVEX Center 和 Center 前端。MySQL 与学校 VPN/ChainMaker 应提前就绪。

> 管理员 token 明文不要写入仓库。Go 的 `backend.yml` 只保存 token 的 SHA-256；Java 通过 `DID_BACKEND_TOKEN` 环境变量获得明文。

## 0. 可选：重新构建 Java

Java 源码或配置有变化时，在 DAVEX 根目录执行：

```bash
cd /Users/dengruotao/Documents/GitHub/DAVEX
sh '/Applications/IntelliJ IDEA.app/Contents/plugins/maven/lib/maven3/bin/mvn' -DskipTests package
```

## 1. 终端一：启动 Go DID backend

```bash
cd /Users/dengruotao/Documents/GitHub/DAVEX/did/backend
env -u GOROOT go run ./cmd/server --config ./configs/backend.yml
```

启动后检查：

```bash
curl http://localhost:8081/api/health
```

Go Swagger：`http://localhost:8081/swagger/`

## 2. 终端二：启动 DAVEX Center

把 `<管理员演示token>` 替换为与 `did/backend/configs/backend.yml` 中 `bootstrap.token_sha256` 对应的明文 token：

```bash
cd /Users/dengruotao/Documents/GitHub/DAVEX/DAVEX_center

export DID_ENABLED=true
export DID_BACKEND_URL=http://localhost:8081
export DID_BACKEND_TOKEN='<管理员演示token>'

java -jar target/davex-center-0.0.1.jar
```

启动后检查：

```bash
curl http://localhost:9900/api/v1/health
curl http://localhost:9900/api/v1/did/health
curl -H 'X-DID-Actor: bootstrap' http://localhost:9900/api/v1/did/ready
```

## 3. 终端三：启动 Center 前端

```bash
cd /Users/dengruotao/Documents/GitHub/DAVEX/DAVEX_ui
npm run dev -- --host 127.0.0.1 --port 8082 --strictPort
```

访问：`http://127.0.0.1:8082/did`

## 停止

分别在三个启动终端中按 `Ctrl+C`。确认端口已经释放：

```bash
lsof -nP -iTCP:8081 -sTCP:LISTEN
lsof -nP -iTCP:9900 -sTCP:LISTEN
lsof -nP -iTCP:8082 -sTCP:LISTEN
```

三个命令都没有输出即表示服务已经停止。
