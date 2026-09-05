# Nexus 排障记录

本文档记录开发过程中遇到的问题、现象、根因与解决方式，方便日后对照排查。

---

## 一、前端 dev server 端口相关

### 1.1 5173 和 5174 端口同时都能访问

**现象**：`npm run dev` 后，`http://localhost:5173` 和 `http://localhost:5174` 都能打开页面。

**根因**：在未停止上一个 dev server 的情况下再次运行 `npm run dev`。Vite 默认 `strictPort: false`，发现 5173 被占用就**自动顺延**到 5174，于是同时跑起了多个 dev server。

**解决**：
1. 杀掉多余进程；
2. 在 [web/vite.config.ts](../web/vite.config.ts) 里加 `strictPort: true`，端口被占时**直接报错退出**，不再静默顺延：

```ts
server: {
  port: 5173,
  strictPort: true,
  proxy: { '/api': { target: 'http://localhost:8080', changeOrigin: true } },
}
```

### 1.2 kill 掉 npm 进程后，5173 仍能访问

**现象**：明明把 `npm` 进程都 kill 了，`http://localhost:5173` 还是打得开。

**根因**：真正监听 5173 的进程**不叫 npm**，而是它的**子进程** `node .../node_modules/.bin/vite`。按名字杀 `npm` 匹配不到那个 `node` 进程；即便杀掉 `npm` 父进程，`node vite` 子进程也会变成**孤儿进程**继续占用端口。

**解决**：停 dev server **按端口杀，别按名字杀**：

```bash
lsof -ti tcp:5173 | xargs kill -9
```

> 正常开发只在一个终端 `npm run dev`，用 `Ctrl+C` 停止即可（会连子进程一起收掉）。上面的命令用于异常情况（终端被强关、进程变孤儿）兜底。

### 1.3 前端报 ENOENT `ChatView.vue`（历史遗留）

**现象**：Vite 报 `ENOENT: no such file or directory, open '/src/views/ChatView.vue'`，但该文件在源码里根本不存在。

**根因**：陈旧的 dev server 持有了过期的模块图缓存。

**解决**：杀掉所有 vite 进程 → 清理 `node_modules/.vite` 缓存 → 重启 dev server → 浏览器硬刷新（Cmd+Shift+R）。

---

## 二、配置迁移到 config.yml

**背景**：原先配置分散在 `.env`（数据库连接串等）。希望统一集中到 `config.yml`。

**做法**：
- 新增 [config.yml](../config.yml)（含密码/密钥，已 `.gitignore` 忽略）与 [config.example.yml](../config.example.yml)（占位模板，提交到仓库）；
- 用已有的 `github.com/goccy/go-yaml` 解析（无需新增依赖）；
- 删除 `.env` / `.env.example`；
- 配置分 `server` / `database` / `llm` 三段。可用环境变量 `CONFIG_FILE` 指定其他路径。

**安全**：`config.yml` 含数据库密码与 LLM 密钥，**切勿提交到仓库**。

---

## 三、接入真实大模型

**目标**：从只有 mock provider，改为可调用真实大模型，且 `base_url` / `api_key` / `model` **按每个 Agent 在页面上配置**。

**做法**：
- 新增通用的 **OpenAI 兼容 provider** [internal/llm/openai.go](../internal/llm/openai.go)，`base_url` 可配，能接 DeepSeek / 通义 / Kimi / 智谱 / OpenAI / 本地 vLLM 等；
- Agent 表新增 `base_url`、`api_key` 两列（model 复用 `model_name`）；
- provider 按 Agent 配置动态构造：`mock` 走本地回显，`openai` 用该 Agent 的连接；字段留空则回退到 `config.yml` 的 `llm` 默认值；
- 前端 Agent 表单：Provider 改为下拉（openai / mock），openai 时显示 Base URL / API Key / Model 输入框。

> ⚠️ 当前 `api_key` 按 Agent **明文存数据库**，且 `GET /agents` 会返回该字段（供表单回填）。仅适用于内部/可信环境；对外部署前应改为加密存储或返回时脱敏。

---

## 四、LLM 调用踩坑（重点）

### 4.1 报错：解析 LLM 响应失败，返回一堆 HTML

**现象**：
```
解析 LLM 响应失败(status=200): <!doctype html> <html lang="en"> ... <title>Ne…
```

**根因**：`base_url` 配错，指向了一个**网页**而不是 API。请求打到 `{base_url}/chat/completions` 被目标站点的 SPA 兜底返回了 `index.html`（状态 200、内容是 HTML）。

本例中 base_url 填的是 `https://nexapi.tech`（少了 `/v1`）：
- `https://nexapi.tech/chat/completions` → 200，返回官网 HTML ❌
- `https://nexapi.tech/v1/chat/completions` → 返回 JSON ✅

**解决**：
1. base_url 填**厂商 API 根地址**，NexAPI 这类中转要带 `/v1`，即 `https://nexapi.tech/v1`；
2. **不要**在 base_url 里带 `/chat/completions`，后端会自动追加；
3. 不要填成 `localhost:5173` / `localhost:8080` 等 Nexus 自身地址。

**代码加固**：provider 现在会检测非 JSON 响应（Content-Type 非 json 或响应体以 `<` 开头），直接报「base_url 配置错误」而不是甩一大段 HTML。见 [internal/llm/openai.go](../internal/llm/openai.go)。

**各厂商 base_url 参考**：

| 厂商 | base_url |
| --- | --- |
| DeepSeek | `https://api.deepseek.com` |
| 通义千问 | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| Kimi | `https://api.moonshot.cn/v1` |
| 智谱 GLM | `https://open.bigmodel.cn/api/paas/v4` |
| OpenAI | `https://api.openai.com/v1` |
| NexAPI（中转） | `https://nexapi.tech/v1` |
| 本地 vLLM/Ollama | `http://localhost:8000/v1` |

### 4.2 报错：Claude 模型拒绝 messages 里的 system 角色

**现象**：
```
LLM 返回错误(status=400): messages.0: use the top-level 'system' parameter
for the initial system prompt; ...
```

**根因**：所选模型是 **Claude 系列**。Anthropic 的规矩是系统提示词必须作为**顶层 `system` 参数**，不能作为 `messages` 数组里的 `system` 角色。而标准 OpenAI 格式恰恰是把 system 放进 messages，经 NexAPI 转发给 Claude 时被拒。只有该 Agent **设置了系统提示词**时才会触发。

**解决**：修改 provider，不再单独发 `system` 角色消息，而是把系统提示词**并入首条消息内容**（用 `\n\n` 分隔）。这样 `messages` 里不含 system 角色，Claude 与 GPT 类模型都兼容。见 [internal/llm/openai.go](../internal/llm/openai.go) 的 `Chat` 方法。

### 4.3 发消息卡住，60 秒后 500

**现象**：前端一直「发送中」，后端日志 `500 | 1m0s | POST .../messages`，正好卡 1 分钟。

**根因**：`1m0s` = provider 的 HTTP 超时（`config.yml` 里 `llm.timeout_seconds: 60`）触发，即上游 LLM 在 60 秒内没返回。多为上游临时过载/冷启动导致挂起（后续重试即恢复）。

**排查手段**：
- 已在 [internal/service/chat_service.go](../internal/service/chat_service.go) 加了服务端错误日志：`LLM 调用失败 provider=... model=... 耗时=...: <err>`，下次失败可直接看到底层原因；
- 直接 curl 上游定位延迟与返回：

```bash
time curl -sS -m 90 https://nexapi.tech/v1/chat/completions \
  -H "Authorization: Bearer <KEY>" \
  -H "Content-Type: application/json" \
  -d '{"model":"<模型名>","messages":[{"role":"user","content":"你好"}],"max_tokens":512}'
```

> Claude 类模型经网关调用时，若行为异常可尝试补上 `max_tokens`（Anthropic 接口强制要求该参数）。

### 4.4 响应慢（约 9 秒），但成功返回 201

**现象**：成功（201），但耗时 ~9.4s。GET 类请求只有几十毫秒。

**根因**：**这是同步（非流式）调用大模型的固有延迟，不是代码问题**。后端会一直阻塞，直到模型把**整段回复全部生成完**才一次性返回。延迟主要来自：① 模型生成整段回复的时间（大头，回复越长越慢）；② 两跳网络（本机→NexAPI→上游，中转再加一层）；③ 模型排队/首 token 延迟（Claude 尤其明显）。

**改善方式**：
- **流式输出（streaming）**（体感正解，✅ 已实现）：模型每生成几个字就推给前端，首字约 1 秒可见，像官网那样逐字冒出。总耗时不变，但**首字延迟**大幅降低，不再干等一大段。
- 小优化：调小 `max_tokens`、让回复更短、换更快的小模型。

**流式实现细节**：

- **Provider 层** [internal/llm/provider.go](../internal/llm/provider.go)：`Provider` 接口新增 `ChatStream(ctx, req, onDelta)`，逐块回调文本片段并返回完整回复；`StreamFunc` 为回调类型。Mock 与 OpenAI 兼容 provider 均实现。
- **OpenAI 兼容 provider** [internal/llm/openai.go](../internal/llm/openai.go)：请求体带 `stream: true`，按 SSE 逐行读 `data: {...}`，取 `choices[].delta.content` 累积，遇 `data: [DONE]` 结束。⚠️ **流式请求不能用 `http.Client{Timeout}`**——那个超时管的是「整个请求生命周期」，长回复会被拦腰截断；改用**不设 Timeout 的 client**，取消交给请求 `ctx`。
- **服务层** [internal/service/chat_service.go](../internal/service/chat_service.go)：抽出公共 `prepare()`（校验归属 / 存 user 消息 / 载历史 / 选 provider）与 `persistAssistant()`，`SendMessage` 与新增的 `SendMessageStream` 复用；流结束后才把**完整回复**一次性落库（中途出错则只保留 user 消息，不落 assistant）。
- **接口层** [internal/handler/message_handler.go](../internal/handler/message_handler.go)：新增 `POST .../messages/stream`，返回 `text/event-stream`。事件：`data:{"delta":"…"}` 逐块、`data:{"done":true,"message":{…}}` 收尾附落库消息、`data:{"error":"…"}` 中途出错。**若在写出任何 delta 前就出错，则退回普通 JSON 错误响应（带正确状态码）**；已开始输出后出错只能作为事件下发（响应头已发出）。另设 `X-Accel-Buffering: no` 关闭 nginx 缓冲。
- **前端** [web/src/api/messages.ts](../web/src/api/messages.ts)：浏览器原生 `EventSource` 只支持 GET，这里用 `fetch` + `ReadableStream` + `TextDecoder` 读 SSE，按空行 `\n\n` 切分事件。[web/src/stores/chat.ts](../web/src/stores/chat.ts) 先推一个空的 assistant 占位气泡，`onDelta` 累加 `content`，结束用落库消息替换；出错时若已有部分内容则标记失败保留，否则移除占位。[web/src/components/MessageBubble.vue](../web/src/components/MessageBubble.vue) 加了「生成中」跳动点/光标；[web/src/components/ChatPanel.vue](../web/src/components/ChatPanel.vue) 监听最后一条消息内容长度变化以跟随滚动。

---

## 五、前端

### 5.1 流式回复已生成完，用户气泡仍卡在「发送中…」

**现象**：AI 回复已经完整逐字显示出来，但用户自己那条消息底下的状态一直是「发送中…」，不消失。

**根因**：Vue 3 响应式的经典坑。`chat` store 里先 `const optimistic = {...}` 建了个**原始对象**，`push` 进 `messages`（一个 `ref` 数组）后，代码仍持有并修改**原始对象引用**（`optimistic.pending = false`）。而 Vue 3 的响应式基于 **Proxy**：只有通过数组返回的**代理对象**去改属性才会触发依赖更新；直接改原始对象绕过了 Proxy 的 `set` 拦截，**不会触发重新渲染**。助手内容之所以能显示，是因为流式过程中别的状态变更顺带触发了几次渲染，那几次读到了原始对象上被改过的最新值——纯属「搭便车」，`pending` 的变更并没有被追踪到，于是永远停在「发送中」。

**解决**：`push` 之后，从数组里取回**响应式代理**再操作，不要改原始闭包引用：

```ts
messages.value.push(optimistic, assistant)
const userRef = messages.value[messages.value.length - 2]      // 代理
const assistantRef = messages.value[messages.value.length - 1] // 代理
// 之后一律改 userRef / assistantRef，onDelta 里也累加 assistantRef.content
```

见 [web/src/stores/chat.ts](../web/src/stores/chat.ts) 的 `send()`。

> 记忆点：**往 `ref`/`reactive` 容器里放对象后，必须通过容器读回来的引用去改**；手里那个「放进去之前的原始对象」改了不算数。

---

## 六、Claude 原生 Provider（顶层 system）

**背景**：见 §4.2——走 OpenAI 兼容 `/chat/completions` 时，system 只能作为消息角色，NexAPI 转发 Claude 又拒收，于是把系统提示词拼进 `messages[0]`。但那样提示词被埋在**最老的一条消息**里，随对话变长离当前问题越来越远，Claude 的遵循度会衰减——远不如 Anthropic 真正的**顶层 `system` 参数**（全程高权重）。

**做法**：新增 [internal/llm/anthropic.go](../internal/llm/anthropic.go)，走 Anthropic 原生 `/v1/messages`，系统提示词作为顶层 `system` 下发。它实现现有 `llm.Provider` 接口（`Chat` + `ChatStream`），所以 service/handler/前端零改动。Agent 页面 Provider 下拉新增 `anthropic`，沿用同一套 base_url/api_key/model，并新增 **Max Tokens**（Anthropic 强制项，留空/0 回退 config 的 `llm.max_tokens`，再回退 4096）。OpenAI 兼容路径（DeepSeek/通义等非 Claude 模型）保持不变。

**关键差异（相对 openai.go）**：

| 项 | OpenAI 兼容 | Anthropic 原生 |
| --- | --- | --- |
| Endpoint 后缀 | 自动补 `/chat/completions` | 自动补 `/v1/messages`（base 已带 `/v1` 则只补 `/messages`） |
| 系统提示词 | 拼进首条消息内容 | 顶层 `system` 参数 |
| 鉴权头 | `Authorization: Bearer` | `x-api-key` + `anthropic-version: 2023-06-01`（并附带 `Bearer` 兼容中转） |
| max_tokens | 可选 | **必填**（Anthropic 强制） |
| 流式 SSE | `data:` 里 `choices[].delta.content`，`[DONE]` 结束 | 事件 `content_block_delta` 的 `delta.text_delta`，`message_stop` 结束 |

**base_url 怎么填（取决于密钥）**：
- 用**真实 Anthropic key** → `https://api.anthropic.com`
- 用 **NexAPI 等中转 key** → 中转的 Anthropic 端点（多为 `.../v1`）；provider 已同时带 `x-api-key` 与 `Bearer`、并自适应 `/v1` 后缀以最大化兼容，但**具体端点建议先 curl 验证一次**：

```bash
curl -sS <base_url>/v1/messages \
  -H "x-api-key: <KEY>" -H "anthropic-version: 2023-06-01" \
  -H "Authorization: Bearer <KEY>" -H "content-type: application/json" \
  -d '{"model":"<claude 模型>","max_tokens":256,"system":"你只能用中文回答","messages":[{"role":"user","content":"hi"}]}'
```

**迁移提示**：原本 `provider=openai` 指向 Claude 模型的 Agent，改成 `provider=anthropic` 即可享受顶层 system；DeepSeek 等非 Claude 模型继续用 `openai`。

### 6.1 报错 429：All providers are saturated; retry shortly

**现象**：
```
LLM 流式调用失败 provider=anthropic model=claude-opus-4-8 耗时=7.7s:
LLM 返回错误(status=429): All providers are saturated; retry shortly (request id: ...)
[GIN] ... | 500 | 7.99s | ... POST .../messages/stream
```

**根因**：**不是代码问题**——这恰恰证明原生 Anthropic 链路是通的（格式被接受、模型被识别、拿到结构化 API 错误并带 NexAPI request id）。`429 saturated` 是**中转网关容量饱和**，opus 等热门大模型尤其容易排不上，属**临时**错误。

**解决**：
1. 过几秒**重发**；
2. 换**没那么挤的模型**（如 `claude-sonnet-*`）验证链路；
3. 需要的话可给 provider 加「429 自动退避重试」（当前未做，按需开发）。

> 附带现象：日志里的 `SLOW SQL >= 200ms`（INSERT messages）是连**远程 PostgreSQL**（部署在公网的实例）的网络延迟，非 bug。

---

## 七、失败消息可重试（改为「成功才落库」）

**需求**：发送失败（如 §6.1 的 429）的消息，允许在气泡上点「重试」重发。

**关键坑**：原先后端在**调用 LLM 之前**就把 user 消息落库了（[chat_service.go](../internal/service/chat_service.go) 的 `prepare`）。若前端「重试」只是再调一次发送接口，会往库里**插入重复的 user 消息**。

**做法**：把落库时机改为**成功才落库**。
- 后端：`prepare` 不再落库，改为加载已有历史并把「本次 user 消息」**仅在内存中**追加到请求末尾；LLM 调通后由新的 `persistExchange` 在**一个事务**里同时写入 user + assistant（显式设时间戳保证 user 早于 assistant）。任何失败都不留痕 → 重试天然不会重复。
- 前端（[stores/chat.ts](../web/src/stores/chat.ts)）：抽出公共的 `runExchange`，`send` 与新增的 `retry(msg)` 共用；`retry` **就地**复用失败的 user 气泡与其后的 assistant 气泡（避免消息跳位），把状态重置为发送中后重跑。
- UI（[MessageBubble.vue](../web/src/components/MessageBubble.vue) / [ChatPanel.vue](../web/src/components/ChatPanel.vue)）：失败的 user 气泡显示「重试」按钮，发送中禁用。

**行为变化提示**：失败的消息现在**不再进数据库**，刷新页面后会消失（未成功的对话不落库）；这与「可原样重试」是配套的取舍。

---

## 八、用户注册登录 + 按用户数据隔离

**需求**：加普通用户名+密码注册登录，并让每个用户只看到/操作自己的数据（不窜台）。

**数据从属链**：`User → Agent(user_id) → Conversation(agent_id) → Message(conversation_id)`。

**隔离设计（关键）**：owner 只挂在**最顶层的 Agent** 上（新增 `Agent.user_id`）。因为会话必属于某 Agent、消息必属于某会话，**只要锁住「Agent 属于谁」这一跳，下层自动隔离**。落地就是把加载 Agent 的查询从 `id = ?` 改成 `id = ? AND user_id = ?`（[chat_service.go](../internal/service/chat_service.go) 的 `prepare`/`History`、[agent_service.go](../internal/service/agent_service.go) 的 `Get`、[conversation_service.go](../internal/service/conversation_service.go) 的 `ensureAgentOwned`）。查不到即 `ErrAgentNotFound`——**既覆盖「不存在」也覆盖「属于他人」（越权）**，对外表现都是 404，不泄露存在性。

**会话机制**：HttpOnly + SameSite=Lax cookie（名 `nexus_session`）+ 服务端 `sessions` 表（token 用 `crypto/rand` 随机串，非 JWT）。选它而非 JWT 的原因：① token 存 HttpOnly cookie，JS 读不到，避免 XSS 窃取；② 服务端有表，**登出即删、可撤销**；③ 不必引入 JWT 库。密码用 `golang.org/x/crypto/bcrypt` 加盐哈希，`PasswordHash` 用 json:"-" 绝不下发。

**前端两个坑**：
1. **`credentials: 'include'`**：fetch 默认不带 cookie，必须显式加。[client.ts](../web/src/api/client.ts) 加了；**流式那处 [messages.ts](../web/src/api/messages.ts) 绕过了 client.ts，要单独加**——漏了它会导致「列表能看、一发消息就 401」。
2. **401 统一处理**：`request()` 收到 401 时 `window.dispatchEvent('nexus:unauthorized')`，[App.vue](../web/src/App.vue) 监听后 `auth.clear()` 跳回登录页（用事件而非直接 import store，避免 client↔store 循环依赖）。

**门禁**：不引 vue-router，沿用 App.vue 条件渲染——`auth.ready` 前显示占位，`!auth.user` 显示 `<LoginView>`，否则渲染三栏主界面。启动时 `auth.fetchMe()` 探测已有 cookie 会话（未登录的 401 静默吞掉）。

### 8.1 迁移坑：给已有数据的表加 `NOT NULL` 列会失败

**现象**：升级后首次 `go run ./cmd/server`，AutoMigrate 报错，类似 `column "user_id" contains null values`。

**根因**：`Agent.user_id` 是 `not null`。Postgres 对**已有数据**的表执行 `ALTER TABLE ADD COLUMN user_id ... NOT NULL` 时，旧行该列为 NULL → 违反约束 → 迁移失败。

**解决（本项目按「清空重来」）**：**先**在库里清掉旧的无归属数据，**再**启动新服务（此时 agents 表为空，加列不再报错）：
```sql
DELETE FROM messages; DELETE FROM conversations; DELETE FROM agents;
```
> 顺序很重要：清空 SQL 要在启动新 server **之前**跑（直接连 Postgres 执行，与服务无关）。若将来不想清空而要保留旧数据，则应改为「先加可空列 → 回填 user_id → 再改 NOT NULL」的三步迁移。

---

## 附：常用排障命令

```bash
# 释放后端端口
lsof -ti tcp:8080 | xargs kill -9

# 释放前端端口（按端口杀，不要按 npm 名字杀）
lsof -ti tcp:5173 | xargs kill -9

# 启动后端（nexus/ 根目录）
go run ./cmd/server

# 启动前端（nexus/web 目录）
npm run dev

# 直接测试上游 LLM 是否可用 / 延迟多少
time curl -sS -m 90 <base_url>/chat/completions \
  -H "Authorization: Bearer <KEY>" -H "Content-Type: application/json" \
  -d '{"model":"<模型名>","messages":[{"role":"user","content":"hi"}],"max_tokens":512}'
```
