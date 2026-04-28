# oc-go-cc

一个 Go CLI 代理，让你在 [Claude Code](https://docs.anthropic.com/en/docs/claude-code) 中使用你的 [OpenCode Go](https://opencode.ai/docs/go/) 订阅。

`oc-go-cc` 位于 Claude Code 与 OpenCode Go 之间，拦截 Anthropic API 请求，将其转换为 OpenAI 格式，再转发到 OpenCode Go 的端点。Claude Code 以为自己在跟 Anthropic 对话——但实际你的请求被路由到了价格实惠的开放模型。

## 为什么要用？

OpenCode Go 以 **$5/月**（之后 $10/月）的价格让你访问强大的开放编程模型。本代理使这些模型无缝适配 Claude Code 的接口——无需补丁、无需 fork，只需设置两个环境变量即可使用。

## 特性

- **透明代理** — Claude Code 发送 Anthropic 格式的请求，代理自动转换为 OpenAI 格式并回传
- **模型路由** — 根据上下文自动将请求路由到不同的模型（默认、推理、长上下文、后台任务）
- **降级链** — 当某个模型请求失败时，自动尝试配置链中的下一个模型
- **熔断器** — 追踪模型健康状态，跳过故障模型以避免延迟飙升
- **实时流式传输** — 完整的 SSE 流式传输，进行 OpenAI → Anthropic 格式的实时转换
- **工具调用** — 正确的 Anthropic tool_use/tool_result ↔ OpenAI function calling 翻译
- **Token 计数** — 使用 tiktoken（cl100k_base）进行精确的 token 计数和上下文阈值检测
- **JSON 配置** — 灵活的配置文件，支持环境变量覆盖和 `${VAR}` 插值
- **后台模式** — 以守护进程方式运行，与终端分离
- **开机自启** — 通过 launchd（macOS）在系统启动时自动运行

## 安装

### Homebrew（macOS 和 Linux）

```bash
brew tap samueltuyizere/tap
brew install oc-go-cc
```

### 从源码构建

```bash
git clone https://github.com/samueltuyizere/oc-go-cc.git
cd oc-go-cc
make build

# 二进制文件位于 bin/oc-go-cc
# 可选安装到 $GOPATH/bin
make install
```

### 下载发布版二进制文件

从 [Releases 页面](https://github.com/samueltuyizere/oc-go-cc/releases) 下载适用于你平台的最新版本：

| 平台                  | 文件                         |
| --------------------- | ---------------------------- |
| macOS（Apple Silicon） | `oc-go-cc_darwin-arm64`      |
| macOS（Intel）         | `oc-go-cc_darwin-amd64`      |
| Linux（x86_64）        | `oc-go-cc_linux-amd64`       |
| Linux（ARM64）         | `oc-go-cc_linux-arm64`       |
| Windows（x86_64）      | `oc-go-cc_windows-amd64.exe` |
| Windows（ARM64）       | `oc-go-cc_windows-arm64.exe` |

```bash
# 示例：macOS Apple Silicon
curl -L -o oc-go-cc https://github.com/samueltuyizere/oc-go-cc/releases/latest/download/oc-go-cc_darwin-arm64
chmod +x oc-go-cc
sudo mv oc-go-cc /usr/local/bin/
```

### 前提条件

- 一个 [OpenCode Go](https://opencode.ai/auth) 订阅和 API 密钥
- Go 1.21+（仅从源码构建时需要）

## 快速开始

### 1. 初始化配置

```bash
oc-go-cc init
```

在 `~/.config/oc-go-cc/config.json` 创建默认配置。

### 2. 设置 API 密钥

```bash
export OC_GO_CC_API_KEY=sk-opencode-你的密钥
```

### 3. 启动代理

```bash
oc-go-cc serve
```

你将看到如下输出：

```
Starting oc-go-cc v0.1.0
Listening on 127.0.0.1:3456
Forwarding to: https://opencode.ai/zen/go/v1/chat/completions

Configure Claude Code with:
  export ANTHROPIC_BASE_URL=http://127.0.0.1:3456
  export ANTHROPIC_AUTH_TOKEN=unused
```

#### 后台运行

在后台运行代理（与终端分离）：

```bash
oc-go-cc serve --background
# 或
oc-go-cc serve -b
```

这将以后台守护进程方式启动服务器并立即返回。日志写入 `~/.config/oc-go-cc/oc-go-cc.log`。

#### 开机自启

在登录时自动启动代理：

```bash
oc-go-cc autostart enable
```

这会在 macOS 上创建一个 launchd plist。禁用：

```bash
oc-go-cc autostart disable
```

查看状态：

```bash
oc-go-cc autostart status
```

### 4. 配置 Claude Code

在另一个终端中（或在运行 `claude` 之前的同一个终端中）：

```bash
export ANTHROPIC_BASE_URL=http://127.0.0.1:3456
export ANTHROPIC_AUTH_TOKEN=unused
```

### 5. 运行 Claude Code

```bash
claude
```

就这样。Claude Code 现在会将所有请求通过 oc-go-cc 路由到 OpenCode Go。

## 工作原理

```
┌─────────────┐   Anthropic API      ┌─────────────┐   OpenAI API         ┌─────────────┐
│  Claude Code ├─────────────────────►│  oc-go-cc    ├─────────────────────►│  OpenCode Go │
│  (CLI)       │  POST /v1/messages  │  (代理)      │  /chat/completions  │  (上游)      │
│              │◄─────────────────────┤              │◄─────────────────────┤              │
└─────────────┘   Anthropic SSE       └─────────────┘   OpenAI SSE          └─────────────┘
```

1. Claude Code 以 [Anthropic Messages API](https://docs.anthropic.com/en/api/messages) 格式发送请求
2. oc-go-cc 解析请求、计算 token 数，并通过路由规则选择模型
3. 请求被转换为 [OpenAI Chat Completions](https://platform.openai.com/docs/api-reference/chat) 格式
4. 转换后的请求发送到 OpenCode Go 的端点
5. 响应（流式或非流式）被转换回 Anthropic 格式
6. Claude Code 收到的响应就像直接来自 Anthropic 一样

### 转换对照表

| Anthropic                                                    | OpenAI                                  |
| ------------------------------------------------------------ | --------------------------------------- |
| `system`（字符串或数组）                                      | `messages[0]` 带 `role: "system"`     |
| `content: [{"type":"text","text":"..."}]`                    | `content: "..."`                        |
| `tool_use` 内容块                                            | `tool_calls` 数组                       |
| `tool_result` 内容块                                         | `role: "tool"` 消息                     |
| `thinking` 内容块                                            | `reasoning_content`                     |
| `stop_reason: "end_turn"`                                    | `finish_reason: "stop"`                 |
| `stop_reason: "tool_use"`                                    | `finish_reason: "tool_calls"`           |
| SSE `message_start` / `content_block_delta` / `message_stop` | SSE `role` / `delta.content` / `[DONE]` |

### DeepSeek V4 推理模式

DeepSeek V4 Pro 和 Flash 通过 OpenCode Go 使用 OpenAI 兼容的 `/chat/completions` 端点。它们支持推理模式和可配置的推理强度。

对于 Claude Code 及其他 agentic 编程工作流，请使用以下配置来设置 DeepSeek V4 模型：

```json
{
  "provider": "opencode-go",
  "model_id": "deepseek-v4-pro",
  "max_tokens": 8192,
  "reasoning_effort": "max",
  "thinking": {
    "type": "enabled"
  }
}
```

`oc-go-cc` 将这些字段作为 OpenAI Chat Completions 参数转发给 OpenCode Go：

- `reasoning_effort`：控制 DeepSeek V4 推理强度（`high` 或 `max`）
- `thinking`：启用或禁用 DeepSeek V4 推理模式

DeepSeek V4 推理响应以 OpenAI `reasoning_content` 形式返回，并被转换为 Anthropic `thinking` 块以供 Claude Code 使用。

## 配置

### 配置文件

位置：`~/.config/oc-go-cc/config.json`

可通过 `OC_GO_CC_CONFIG` 环境变量覆盖。

### 完整配置参考

```json
{
  "api_key": "${OC_GO_CC_API_KEY}",
  "host": "127.0.0.1",
  "port": 3456,

  "models": {
    "default": {
      "provider": "opencode-go",
      "model_id": "kimi-k2.6",
      "temperature": 0.7,
      "max_tokens": 4096
    },
    "background": {
      "provider": "opencode-go",
      "model_id": "qwen3.5-plus",
      "temperature": 0.5,
      "max_tokens": 2048
    },
    "think": {
      "provider": "opencode-go",
      "model_id": "glm-5.1",
      "temperature": 0.7,
      "max_tokens": 8192
    },
    "long_context": {
      "provider": "opencode-go",
      "model_id": "minimax-m2.7",
      "temperature": 0.7,
      "max_tokens": 16384,
      "context_threshold": 60000
    },
    "deepseek_v4_max": {
      "provider": "opencode-go",
      "model_id": "deepseek-v4-pro",
      "temperature": 0.1,
      "max_tokens": 8192,
      "reasoning_effort": "max",
      "thinking": {
        "type": "enabled"
      }
    }
  },

  "fallbacks": {
    "default": [
      { "provider": "opencode-go", "model_id": "glm-5" },
      { "provider": "opencode-go", "model_id": "qwen3.6-plus" }
    ],
    "think": [{ "provider": "opencode-go", "model_id": "glm-5" }],
    "long_context": [{ "provider": "opencode-go", "model_id": "minimax-m2.5" }]
  },

  "opencode_go": {
    "base_url": "https://opencode.ai/zen/go/v1/chat/completions",
    "timeout_ms": 300000
  },

  "logging": {
    "level": "info",
    "requests": true
  }
}
```

### 环境变量

环境变量会覆盖配置文件中的值。配置文件的值也支持 `${VAR}` 插值。

| 变量                    | 描述                                     | 默认值                                             |
| ----------------------- | ---------------------------------------- | -------------------------------------------------- |
| `OC_GO_CC_API_KEY`      | OpenCode Go API 密钥（**必填**）          | —                                                  |
| `OC_GO_CC_CONFIG`       | 自定义配置文件路径                        | `~/.config/oc-go-cc/config.json`                   |
| `OC_GO_CC_HOST`         | 代理监听地址                              | `127.0.0.1`                                        |
| `OC_GO_CC_PORT`         | 代理监听端口                              | `3456`                                             |
| `OC_GO_CC_OPENCODE_URL` | OpenCode Go API 端点                      | `https://opencode.ai/zen/go/v1/chat/completions`   |
| `OC_GO_CC_LOG_LEVEL`    | 日志级别：`debug`、`info`、`warn`、`error` | `info`                                             |

### 模型路由

代理会根据上下文大小和内容分析，自动检测请求类型并路由到合适的模型：

| 场景         | 触发条件                                             | 模型          | 原因                                        |
| ------------ | ---------------------------------------------------- | ------------- | ------------------------------------------- |
| **长上下文** | >60K token                                           | MiniMax M2.7  | 1M 上下文窗口，而其他模型为 128-256K        |
| **复杂任务** | 系统提示中包含 "architect"、"refactor"、"complex"      | GLM-5.1       | 最佳推理与架构理解能力                      |
| **推理任务** | 系统提示中包含 "think"、"plan"、"reason"               | GLM-5         | 良好推理能力，比 GLM-5.1 更便宜             |
| **后台任务** | "read file"、"grep"、"list directory"                  | Qwen3.5 Plus  | 最便宜（~10K 请求/5小时），非常适合简单操作 |
| **默认**     | 其他所有请求                                          | Kimi K2.6     | 质量与成本的最佳平衡（~1.8K 请求/5小时）    |

**📖 详见 [MODELS.md](MODELS.md)，了解详细的模型能力、成本及路由建议。**

DeepSeek V4 用户可以将任意场景模型设置为 `deepseek-v4-pro` 或 `deepseek-v4-flash`。要使用确定性最大推理，请在对应场景的模型配置和降级条目中添加 `reasoning_effort: "max"` 和 `thinking: {"type":"enabled"}`。

#### 路由详解：

| 场景         | 触发条件                                                                      | 配置键                 | 默认模型        |
| ------------ | ----------------------------------------------------------------------------- | ---------------------- | --------------- |
| **默认**     | 标准对话                                                                      | `models.default`       | `kimi-k2.6`     |
| **推理任务** | 系统提示包含 "think"、"plan"、"reason"；或含有 thinking 内容块                  | `models.think`         | `glm-5.1`       |
| **长上下文** | Token 数超过 `context_threshold`                                              | `models.long_context`  | `minimax-m2.7`  |
| **后台任务** | 文件读取、目录列表、grep 模式匹配                                              | `models.background`    | `qwen3.5-plus`  |

路由优先级：**长上下文** → **推理任务** → **后台任务** → **默认**

### 降级链

当模型请求失败（网络错误、速率限制、服务器错误）时，代理会尝试降级链中的下一个模型：

```
主模型 → 降级模型 1 → 降级模型 2 → ... → 错误（全部失败）
```

每个模型还有一个**熔断器**，用于追踪连续失败次数。连续失败 3 次后，熔断器断开，该模型将被跳过 30 秒，之后再次测试（半开状态）。

### 可用模型

详见 [MODELS.md](MODELS.md) 中的**模型能力、成本和路由建议**。

快速参考：

| 模型 ID          | 质量   | 上下文  | 成本（请求/5小时） | 最佳用途                              |
| ---------------- | ------ | ------- | ------------------ | ------------------------------------- |
| `glm-5.1`        | ★★★★★   | 200K    | ~880               | 复杂架构、困难任务                    |
| `glm-5`          | ★★★★☆   | 200K    | ~1,150             | 高质量编码、重构                      |
| `kimi-k2.6`      | ★★★★★   | 256K    | ~1,850             | **默认选择** - 最佳平衡               |
| `kimi-k2.5`      | ★★★★☆   | 256K    | ~1,850             | 降级备用 - 质量可靠                   |
| `mimo-v2-pro`    | ★★★★☆   | 128K    | ~1,290             | 代码补全、代码生成                    |
| `mimo-v2-omni`   | ★★★☆☆   | 256K    | ~2,150             | 快速原型开发                          |
| `qwen3.6-plus`   | ★★★☆☆   | 128K    | ~3,300             | 高性价比通用编码                      |
| `minimax-m2.7`   | ★★★☆☆   | **1M**  | ~3,400             | **长上下文专用**                      |
| `minimax-m2.5`   | ★★☆☆☆   | **1M**  | ~6,300             | 经济型长上下文                        |
| `deepseek-v4-pro` | ★★★★★ | **1M** | 不定               | Agentic 编码、最大推理、长上下文      |
| `deepseek-v4-flash` | ★★★★☆ | **1M** | 不定              | 快速 Agent 任务、后台/子代理工作      |
| `qwen3.5-plus`   | ★★☆☆☆   | 128K    | ~10,200            | **最便宜** - 后台任务                 |

> **💡 提示：** 成本列显示每 5 小时（$12）大约可发出的请求数。Qwen3.5 Plus 的请求数比 GLM-5.1 多约 10 倍！

> **⚠️ 重要：** MiniMax M2.5 和 M2.7 原生使用 **Anthropic 兼容**的 `/v1/messages` 端点。DeepSeek V4 Pro 和 Flash 使用 **OpenAI 兼容**的 `/chat/completions` 端点，并支持 `reasoning_effort` 和 `thinking` 以实现最大推理模式。详见 [MODELS.md](MODELS.md)。

## CLI 命令

```
oc-go-cc serve              启动代理服务器
oc-go-cc serve -b           在后台启动（与终端分离）
oc-go-cc serve --port 8080  在自定义端口上启动
oc-go-cc serve --config /path/to/config.json  使用自定义配置
oc-go-cc stop               停止正在运行的代理服务器
oc-go-cc status             检查代理是否运行中
oc-go-cc autostart enable   启用在登录时自动启动
oc-go-cc autostart disable  禁用在登录时自动启动
oc-go-cc autostart status   检查自动启动状态
oc-go-cc init               创建默认配置文件
oc-go-cc validate           验证配置文件
oc-go-cc models             列出可用的 OpenCode Go 模型
oc-go-cc --version          显示版本号
```

## API 端点

代理暴露 Claude Code 所期望的以下端点：

| 方法   | 路径                          | 描述                                |
| ------ | ----------------------------- | ----------------------------------- |
| `POST` | `/v1/messages`                | 主聊天端点（Anthropic 格式）        |
| `POST` | `/v1/messages/count_tokens`   | Token 计数                          |
| `GET`  | `/health`                     | 健康检查                            |

## 故障排查

### "invalid request body" 错误

这意味着代理无法解析来自 Claude Code 的请求。启用 debug 日志查看原始请求：

```json
{ "logging": { "level": "debug" } }
```

或设置环境变量：

```bash
export OC_GO_CC_LOG_LEVEL=debug
```

### "all models failed" 错误

降级链中的所有模型都返回了错误。请检查：

1. API 密钥是否有效：`oc-go-cc validate`
2. 是否已超出[使用限制](https://opencode.ai/auth)
3. OpenCode Go 服务是否可达：`curl -H "Authorization: Bearer $OC_GO_CC_API_KEY" https://opencode.ai/zen/go/v1/models`

### 连接被拒绝

确保代理正在运行：

```bash
oc-go-cc status
```

并且 Claude Code 指向正确的地址：

```bash
echo $ANTHROPIC_BASE_URL  # 应该是 http://127.0.0.1:3456
```

### 流式传输不工作

代理实时将 OpenAI SSE 转换为 Anthropic SSE。如果流式传输出现问题：

1. 将日志级别设置为 `debug` 以查看原始 SSE 数据块
2. 检查是否有代理或防火墙缓冲了连接
3. 先尝试非流式请求，验证模型是否正常工作

### Debug 模式

使用 debug 级别运行以获取最详细的日志：

```bash
OC_GO_CC_LOG_LEVEL=debug oc-go-cc serve
```

这会记录：

- 来自 Claude Code 的原始 Anthropic 请求体
- 发送到 OpenCode Go 的转换后 OpenAI 请求
- 收到的原始 OpenAI 响应
- 流式传输期间的 SSE 事件

## 架构

```
cmd/oc-go-cc/main.go           CLI 入口（cobra 命令）
internal/
├── config/
│   ├── config.go               配置类型定义
│   └── loader.go               JSON 加载、环境变量覆盖、${VAR} 插值
├── router/
│   ├── model_router.go         基于场景的模型选择
│   ├── scenarios.go            场景检测（默认/推理/长上下文/后台任务）
│   └── fallback.go            降级处理器，带熔断器
├── server/
│   └── server.go               HTTP 服务器配置、优雅关闭、PID 管理
├── handlers/
│   ├── messages.go             POST /v1/messages 处理器（流式 + 非流式）
│   └── health.go               健康检查和 token 计数端点
├── transformer/
│   ├── request.go              Anthropic → OpenAI 请求转换
│   ├── response.go             OpenAI → Anthropic 响应转换
│   └── stream.go               实时 SSE 流式转换
├── client/
│   └── opencode.go             OpenCode Go HTTP 客户端
└── token/
    └── counter.go              Tiktoken token 计数器（cl100k_base）
pkg/types/
├── anthropic.go                Anthropic API 类型（多态 system/content 字段）
└── openai.go                   OpenAI API 类型
configs/
└── config.example.json         示例配置
```

### 关键设计决策

- **多态字段处理**：Anthropic 的 `system` 和 `content` 字段既可以接受字符串也可以接受数组。我们使用 `json.RawMessage` 配合访问器方法（`SystemText()`、`ContentBlocks()`）来正确处理这两种格式。
- **实时流式代理**：SSE 事件在传输过程中进行转换，而非缓冲。这意味着 Claude Code 可以实时看到来自 OpenCode Go 的响应。
- **每个模型独立的熔断器**：每个模型都有独立的熔断器。连续 3 次失败后，该模型将被跳过 30 秒，之后再次测试。
- **环境变量插值**：诸如 `"${OC_GO_CC_API_KEY}"` 的配置值在加载时被解析，因此你永远不需要在配置文件中存储密钥。

## 开发

```bash
# 构建（版本号从 git 自动检测）
make build

# 开发模式运行
make run

# 运行测试（带竞态检测）
make test

# 运行 go vet
make vet

# 清理构建产物
make clean

# 安装到 $GOPATH/bin
make install

# 构建跨平台发布二进制文件
make dist
```

## 许可证

MIT