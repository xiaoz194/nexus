# Nexus

Nexus 是一个 Agent 平台后端服务（MVP）。支持创建 Agent、为 Agent 创建对话会话、在会话中发送消息并获取 AI 回复、追溯历史消息。**核心保证：不同 Agent 之间的会话与消息严格隔离，无法互访。**

LLM 支持两种 Provider：`mock`（本地回显，零依赖）和 `openai`（OpenAI 兼容接口，可接 DeepSeek / 通义千问 / Kimi / 智谱 / 本地 vLLM 等真实大模型，仅需换 `base_url`）。

## 技术栈

- Go 1.22+（本仓库使用 1.25）
- Gin（Web 框架）
- GORM（ORM）
- PostgreSQL，驱动 `gorm.io/driver/postgres`（默认）；开发也可切换到 SQLite（纯 Go 驱动 `github.com/glebarez/sqlite`，无需 CGO）
- `github.com/google/uuid`（主键）
- `github.com/goccy/go-yaml`（`config.yml` 配置）

## 目录结构

```
nexus/
├── cmd/server/main.go          # 入口：加载配置、初始化数据库、注册路由
├── internal/
│   ├── config/                 # config.yml 配置加载
│   ├── database/               # 数据库连接与 AutoMigrate
│   ├── model/                  # GORM 模型：Agent / Conversation / Message
│   ├── handler/                # HTTP 层：解析请求、调用 service、返回响应
│   ├── service/                # 业务逻辑、隔离校验、事务、调用 LLM
│   └── llm/                    # LLM 抽象：Provider 接口 + Mock / OpenAI 兼容 Provider
├── web/                        # 前端（Vue 3 + Vite + TS + Tailwind），见 web/README.md
├── config.example.yml
├── Makefile
└── README.md
```

各层职责：`handler` 只做 HTTP 编解码；`service` 承载业务逻辑与数据隔离校验；`model` 是数据模型；`llm` 是大模型调用抽象。

## 配置

全部配置集中在 `config.yml`（含密码与密钥）。复制模板后按需修改：

```bash
cp config.example.yml config.yml
```

| 配置项 | 默认值 | 说明 |
| --- | --- | --- |
| `server.port` | `8080` | HTTP 监听端口 |
| `database.type` | `postgres` | 数据库类型：`postgres` 或 `sqlite` |
| `database.dsn` | — | 连接串。postgres 用 URL 形式；sqlite 填文件路径（如 `nexus.db`） |
| `llm.base_url` | `https://api.deepseek.com` | OpenAI 兼容服务的 API 根地址，换厂商只改这一行（会自动追加 `/chat/completions`） |
| `llm.api_key` | `""` | 大模型 API Key。**留空则只有 `mock` provider 可用** |
| `llm.model` | `deepseek-chat` | 默认模型；Agent 未填 `model_name` 时使用 |
| `llm.timeout_seconds` | `60` | LLM 请求超时（秒） |

> 配置文件 `config.yml` 含密码/密钥，已被 `.gitignore` 忽略，**不要提交到仓库**（可改环境变量 `CONFIG_FILE` 指向其他路径）。数据默认存储在 PostgreSQL 的 `nexus` 库，首次启动会自动 `AutoMigrate` 建表。
>
> Postgres URL 形式：`postgres://<user>:<password>@<host>:5432/<db>?sslmode=disable`。全新 Docker Postgres 未配置 SSL，需 `sslmode=disable`。

### 接入真实大模型

大模型连接**按 Agent 在页面上配置**：新建/编辑 Agent 时选 **Provider = `openai`**，填 **Base URL / API Key / Model** 即可。采用 OpenAI 兼容协议，绝大多数厂商开箱即用，换厂商只改 Base URL：

| 厂商 | Base URL | 示例 Model |
| --- | --- | --- |
| DeepSeek | `https://api.deepseek.com` | `deepseek-chat` |
| 通义千问 | `https://dashscope.aliyuncs.com/compatible-mode/v1` | `qwen-plus` |
| Kimi | `https://api.moonshot.cn/v1` | `moonshot-v1-8k` |
| 智谱 GLM | `https://open.bigmodel.cn/api/paas/v4` | `glm-4-flash` |
| OpenAI | `https://api.openai.com/v1` | `gpt-4o-mini` |
| 本地 vLLM/Ollama | `http://localhost:8000/v1` | 你部署的模型名 |

- Provider = `mock` 则走本地回显，无需任何连接配置。
- `config.yml` 的 `llm` 段是**可选的全局回退默认值**：某 Agent 的 Base URL / API Key / Model 留空时，用 config.yml 里对应的值兜底；都不填且无默认 `api_key` 时，发消息返回 400 提示。`timeout_seconds` 只在 config.yml 配置。
- openai Agent 缺 `api_key` → 发消息返回 `400`；Provider 填了 `mock`/`openai` 以外的值 → `501`。

> ⚠️ 安全提示：当前 `api_key` 按 Agent 明文存于数据库，且 `GET /agents` 会返回该字段（供前端表单回填）。仅适用于内部/可信环境；对外部署前应改为加密存储或返回时脱敏。

## 启动

需要分别启动后端和前端，建议开两个终端。前端在开发模式下通过 Vite 代理把 `/api` 转发到后端，无需处理跨域。

### 1. 后端（端口 8080）

```bash
# 在 nexus/ 根目录
cp config.example.yml config.yml   # 首次：填入 PostgreSQL 连接串与 LLM api_key
go run ./cmd/server                # 或 make run
```

启动后监听 `http://localhost:8080`。数据存储在 `database.dsn` 指向的 PostgreSQL `nexus` 库，重启后数据仍然存在；首次启动会自动 `AutoMigrate` 建表。

### 2. 前端（端口 5173）

```bash
# 在 nexus/web 目录
cd web
npm install               # 首次
npm run dev
```

打开 Vite 提示的地址（默认 http://localhost:5173）即可使用。

> 先启动后端再打开前端；否则前端调用接口会提示「无法连接到服务器」。前端详细说明见 [web/README.md](web/README.md)。

### 常用命令

后端（在 `nexus/` 根目录）：

```bash
make run     # go run ./cmd/server
make build   # go build ./...
make fmt     # gofmt -w .
make tidy    # go mod tidy
```

前端（在 `nexus/web` 目录）：

```bash
npm run dev       # 开发服务器（含 API 代理）
npm run build     # 类型检查 + 打包到 dist/
npm run preview   # 预览构建产物（不含 API 代理）
```

## API

统一前缀：`/api/v1`。错误响应统一为 `{"error": "..."}`。

### Agent

| Method | Path | 说明 |
| --- | --- | --- |
| POST | `/api/v1/agents` | 创建 Agent（`201`） |
| GET | `/api/v1/agents` | Agent 列表 |
| GET | `/api/v1/agents/:agent_id` | Agent 详情 |
| PUT | `/api/v1/agents/:agent_id` | 更新 Agent |
| DELETE | `/api/v1/agents/:agent_id` | 删除 Agent 及其全部会话和消息（`204`） |

创建请求体（除 `name` 外均可选）：

```json
{
  "name": "翻译官",
  "description": "中英翻译专家",
  "system_prompt": "你是一个资深的翻译官。",
  "model_provider": "mock",
  "model_name": "mock",
  "temperature": 0.7
}
```

### Conversation

| Method | Path | 说明 |
| --- | --- | --- |
| POST | `/api/v1/agents/:agent_id/conversations` | 创建会话（`201`），`title` 可省略 |
| GET | `/api/v1/agents/:agent_id/conversations` | 该 Agent 下所有会话 |
| DELETE | `/api/v1/agents/:agent_id/conversations/:conversation_id` | 删除会话及其消息（`204`） |

### Message

| Method | Path | 说明 |
| --- | --- | --- |
| POST | `/api/v1/agents/:agent_id/conversations/:conversation_id/messages` | 发送消息，返回助手回复（`201`） |
| GET | `/api/v1/agents/:agent_id/conversations/:conversation_id/messages` | 按 `created_at` 升序返回全部历史消息 |

发送消息请求体：

```json
{ "content": "你好，请翻译这段话" }
```

响应为 assistant 消息对象：

```json
{
  "id": "uuid",
  "conversation_id": "uuid",
  "role": "assistant",
  "content": "Echo: 你好，请翻译这段话",
  "created_at": "2026-01-01T00:00:00Z"
}
```

## 错误码

| 状态码 | 场景 |
| --- | --- |
| 400 | 请求参数校验失败（如缺少 `name`、`content`） |
| 404 | Agent 不存在；会话不存在，或会话不属于该 Agent |
| 500 | LLM Provider 调用失败等内部错误 |
| 501 | Agent 的 `model_provider` 非 `mock`（保留给后续扩展） |

> 隔离保证：当 `conversation_id` 不属于路径中的 `agent_id` 时一律返回 `404`，不泄露数据是否存在。所有会话查询在数据库层面同时以 `id = ? AND agent_id = ?` 双重条件过滤。

## 快速验证

```bash
BASE=http://localhost:8080/api/v1

# 创建 Agent
AID=$(curl -s -X POST $BASE/agents -d '{"name":"翻译官"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')

# 创建会话
CID=$(curl -s -X POST $BASE/agents/$AID/conversations -d '{"title":"demo"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')

# 发送消息 -> 返回 Echo 回复
curl -s -X POST $BASE/agents/$AID/conversations/$CID/messages -d '{"content":"Hello"}'

# 查看历史
curl -s $BASE/agents/$AID/conversations/$CID/messages
```

## MVP 范围之外

不包含：用户系统 / 鉴权、多 Agent 协作、流式输出（SSE）、工具调用、前端界面、记忆机制、消息编辑删除。
