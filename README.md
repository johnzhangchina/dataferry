# DataFerry

**不改代码，摆渡任意两个系统之间的数据。**

DataFerry 是一个轻量级 API 胶水平台。接收任意 Webhook，通过可视化字段映射转换数据格式，转发到任意目标 API。两边系统零侵入。

## 适用场景

- 企业内部子系统之间的数据对接
- 小众 / 垂直行业 SaaS 之间的集成
- 自建系统 / 私有化部署系统的互通

不做预置连接器，不绑定具体平台。用户自己配置两端，DataFerry 只做中间的转换引擎。

## 核心功能

- **接收** — 给每个流程生成专属 Webhook URL，接收任意 JSON 推送
- **映射** — 可视化配置字段映射，支持直接映射、固定值、字符串拼接、数值运算
- **转发** — 按配置的目标 API 格式发出请求，支持重试和超时控制
- **从 JSON 生成** — 粘贴源/目标 JSON 示例，自动提取字段并智能匹配
- **签名验证** — HMAC-SHA256 校验 Webhook 请求来源
- **执行日志** — 完整记录每次转发的请求/响应，支持分页和筛选
- **失败重发** — 转发失败的记录可一键重发或批量重发
- **单文件部署** — 前端打包进 Go 二进制，下载即用

## 快速开始

### 方式一：下载二进制（推荐）

从 [Releases](../../releases) 页面下载对应平台的压缩包：

```bash
# Linux
curl -LO https://github.com/johnzhangchina/dataferry/releases/latest/download/dataferry-linux-amd64.tar.gz
tar xzf dataferry-linux-amd64.tar.gz
./dataferry-linux-amd64

# macOS (Apple Silicon)
curl -LO https://github.com/johnzhangchina/dataferry/releases/latest/download/dataferry-darwin-arm64.tar.gz
tar xzf dataferry-darwin-arm64.tar.gz
./dataferry-darwin-arm64
```

### 方式二：Docker

```bash
# 使用预构建镜像
docker run -d -p 8080:8080 -v dataferry-data:/app/data ghcr.io/johnzhangchina/dataferry:latest

# 或自行构建
docker build -t dataferry .
docker run -d -p 8080:8080 -v dataferry-data:/app/data dataferry
```

### 方式三：从源码编译

```bash
# 需要 Go 1.22+ 和 Node.js 18+
make build
./dataferry
```

打开 http://localhost:8080 即可使用。

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DATAFERRY_PORT` | `8080` | 服务端口 |
| `DATAFERRY_DB` | `dataferry.db` | SQLite 数据库路径 |
| `DATAFERRY_PASSWORD` | (空) | 管理界面密码，不设则不需要登录 |

### 密码保护

设置 `DATAFERRY_PASSWORD` 环境变量即可启用管理界面密码保护：

```bash
DATAFERRY_PASSWORD=your-secret ./dataferry
```

- 管理界面和 API 需要登录后才能访问
- Webhook 接收端点和健康检查不受影响
- 不设密码则管理界面对所有人开放

## 使用指南

### 1. 创建流程

打开管理界面，点击「+ 新建」，填写流程名称。

### 2. 配置目标 API

在「目标 API」卡片中填写：
- 请求方法和 URL
- 需要的请求头（如认证信息）
- 超时时间和重试策略

### 3. 配置字段映射

**手动添加：** 点击「+ 添加」逐条配置映射规则。

**从 JSON 生成：** 点击「从 JSON 生成」，粘贴源 Webhook 的 JSON 示例和目标 API 的 JSON 格式，系统自动匹配同名字段。

支持四种映射类型：

| 类型 | 说明 | 示例 |
|------|------|------|
| 字段取值 | 从源 JSON 取值 | `data.user_name` → `username` |
| 固定值 | 写入固定值 | `"子公司A"` → `source` |
| 字符串拼接 | 模板替换 | `{{first}} {{last}}` → `fullname` |
| 数值运算 | 简单计算 | `price * 100` → `amount_cent` |

支持多层嵌套路径：`data.user.contact.email` → `profile.email`

### 4. 测试

在「在线测试」卡片中粘贴测试 JSON，点击「发送测试」，查看映射预览和实际响应。

### 5. 使用 Webhook URL

将流程的 Webhook URL 配置到源系统中。当源系统推送数据时，DataFerry 会自动转换并转发到目标 API。

## API 参考

### Webhook

```
POST /webhook/{path}
Content-Type: application/json

{源系统推送的 JSON}
```

如果配置了签名密钥，需要在请求头中添加：
```
X-Signature-256: sha256={HMAC-SHA256 十六进制}
```

### 管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/flows` | 流程列表（含最近执行状态） |
| `POST` | `/api/flows` | 创建流程 |
| `GET` | `/api/flows/{id}` | 流程详情 |
| `PUT` | `/api/flows/{id}` | 更新流程 |
| `DELETE` | `/api/flows/{id}` | 删除流程 |
| `GET` | `/api/flows/{id}/logs?page=1&size=20&status=success` | 执行日志 |
| `POST` | `/api/flows/{id}/logs/{logId}/retry` | 重发失败记录 |
| `GET` | `/health` | 健康检查 |

## 开发

```bash
# 安装依赖
cd web/frontend && npm install && cd ../..

# 开发模式：后端 + 前端热更新
go run ./cmd/dataferry &
cd web/frontend && npm run dev

# 运行测试
make test

# 查看所有 make 命令
make help
```

## 技术栈

- **后端：** Go（标准库 net/http + SQLite）
- **前端：** React + TypeScript + Vite
- **部署：** 单二进制（go:embed 打包前端）/ Docker

## 许可证

MIT
