# Nexus Web

Nexus 平台的前端界面。三栏式布局（Agents → 会话 → 聊天），覆盖 MVP 全部核心流程：管理 Agent、管理会话、收发消息、查看历史。UI 层面严格贴合后端「Agent 之间会话隔离」的语义。

## 技术栈

- Vue 3 + `<script setup>` + TypeScript
- Vite（构建 / dev server）
- Pinia（状态管理）
- Tailwind CSS（样式）

## 前置条件

先启动 Go 后端（监听 `:8080`）：

```bash
cd ..           # 到 nexus/ 根目录
go run ./cmd/server
```

## 开发运行

```bash
npm install
npm run dev
```

打开 Vite 提示的地址（默认 http://localhost:5173）。开发期通过 Vite 代理把 `/api` 转发到 `http://localhost:8080`，**无需处理 CORS，后端零改动**（见 [vite.config.ts](vite.config.ts)）。

## 构建

```bash
npm run build      # vue-tsc 类型检查 + vite 打包，产物在 dist/
npm run preview    # 本地预览构建产物
```

> 注意：`npm run preview` 不含 API 代理。若要在生产直接托管 `dist/`，可让 Go 后端用 `StaticFS` 提供静态文件（本 MVP 默认未做）。

## 配置

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `VITE_API_BASE` | `/api/v1` | API 基地址。留空即走 Vite 代理；也可指向完整后端地址。 |

## 目录结构

```
web/
├── vite.config.ts          # /api 代理配置
├── src/
│   ├── App.vue             # 三栏布局 + store 联动（切 Agent 重置会话/聊天）
│   ├── types.ts            # Agent / Conversation / Message 类型（对齐后端 JSON）
│   ├── api/                # fetch 封装 + 各资源接口
│   ├── stores/             # Pinia：agents / conversations / chat / toast
│   └── components/         # AgentList / ConversationList / ChatPanel / ...
```

## 交互要点

- **发送消息**：后端只返回 assistant 一条消息，前端先乐观渲染 user 气泡，成功后追加助手回复；失败标红并弹 Toast（如后端返回 501/500）。
- **隔离**：所有会话/消息请求都带当前选中的 `agentId`；切换 Agent 时清空会话与聊天选中态。
- **错误处理**：统一解包后端 `{"error": "..."}` 并以 Toast 展示。
