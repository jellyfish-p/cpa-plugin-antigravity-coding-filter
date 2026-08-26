# CPA Antigravity Coding Filter 调研记录

调研日期：2026-08-12

目标：评估 `antigravity-coding-filter` 插件如何在 CPA 反代场景中安全改写 `pi`、`Kimi`、`Codex`、`Gemini`、`Claude` 等客户端身份标识为 `Antigravity`，避免上游封号，同时不影响本机正常使用。

## 当前 CPA / 插件状态

- CPA 容器：`cpamp-cli-proxy-api-1`
- CPA 镜像：`eceasy/cli-proxy-api:latest`
- CPA 端口：`8317`
- 管理面板容器：`cpamp-cpa-manager-plus-1`
- 管理面板端口：`18317`
- 插件文件：`plugins/linux/amd64/antigravity-coding-filter-v0.2.0.so`
- 插件版本：`0.2.0`
- 插件状态：`registered=true`、`enabled=true`、`effective_enabled=true`
- 当前模式：`rewrite`
- 当前默认关键词：`use_default_keywords: true`

插件官方源码：

- `https://github.com/jellyfish-p/cpa-plugin-antigravity-coding-filter`
- 当前 main 分支版本：`0.2.1`
- 本机运行版本：`0.2.0`

## 插件当前能力边界

插件当前只扫描 JSON 请求体中键名严格为 `system` 的字段。

它不会处理：

- `messages[]` 中 `role: "system"` 的消息
- OpenAI Responses 的 `instructions`
- Gemini 的 `systemInstruction`
- HTTP 请求头
- 工具声明 `tools[]`
- 模型名、工具名、认证头

`custom_mappings` 语义：

- `rewrite` 模式：左侧关键词替换为右侧值。
- `block` 模式：只使用左侧关键词拦截，右侧值被忽略。
- 匹配：不区分大小写。
- 去重：相同 key 后定义优先。
- 匹配顺序：按 key 长度降序，长关键词优先。

CPA 插件 ABI 已确认：

- `RequestInterceptRequest` 包含 `Headers`
- `RequestInterceptResponse` 可返回 `Headers`
- 可以覆盖请求头，也可以 `ClearHeaders`
- 因此改写 header 不需要改 CPA，只需要改插件。

## 本机已安装版本

- `agy 1.1.11`
- `pi 0.82.1`
- `kimi 0.35.0`
- `codex-cli 0.147.0`
- `gemini 0.53.1`
- `claude 2.1.226`

## agy / Antigravity CLI

来源：本机 `/home/ubuntu/.local/bin/agy` 二进制 strings、官方文档、Codelabs。

system prompt 形态：

- Gemini API：`systemInstruction`
- 内部 protobuf：`CortexStepSystemMessage`
- 内部 system prompt 模板：`function_call.tmpl`、`planning_mode.tmpl`、`slash_commands.tmpl`

已确认身份文本：

- `You are Antigravity Agent, a powerful agentic AI coding assistant designed by the Google engineering team.`
- `You are a subagent of Antigravity, a powerful agentic AI coding assistant designed by the Google Deepmind team working on Advanced Agentic Coding.`

已确认工具名约 21 个：

- `create_file`
- `delete_file`
- `list_dir`
- `execute_command`
- `code_search`
- `web_search`
- `google_search`
- `browser_get_dom`
- `capture_browser_screenshot`
- `execute_browser_javascript`
- `browser_mouse_down`
- `browser_mouse_up`
- `browser_scroll`
- `ask_question`
- `manage_inbox`
- `send_message`
- `finish`
- `HTTP_REQUEST`
- 遗留名：`write_to_file`
- 遗留名：`list_directory`
- 遗留名：`run_command`

不可全局替换：

- `Cascade`：内部 protobuf / 状态机代号
- `jetski` / `Jetski`：内部代码路径和企业标识
- `codeium` / `Codeium`：底层 SDK 标识
- `gemini` / `Gemini`：模型族名
- `google_search`：工具名
- `CloudCode` / `cloudcode`：后端服务名
- `businessaicode`：API 域名
- 工具名集合

## pi 0.82.1

源码路径：`/home/ubuntu/.nvm/versions/node/v24.18.0/lib/node_modules/@earendil-works/pi-coding-agent`

system prompt 注入方式：

- Anthropic API key：顶层 `system`
- Anthropic OAuth：顶层 `system`，且首块硬编码 `You are Claude Code, Anthropic's official CLI for Claude.`
- OpenAI Chat：`messages[0].role = system` 或 `developer`
- OpenAI Responses：顶层 `instructions`
- Codex Responses：顶层 `instructions`
- Gemini：`systemInstruction`
- Vertex：`systemInstruction`
- Bedrock：顶层 `system`
- Mistral：`messages[0].role = system`

内置工具 7 个：

- `read`
- `bash`
- `edit`
- `write`
- `grep`
- `find`
- `ls`

默认启用 4 个：

- `read`
- `bash`
- `edit`
- `write`

请求头指纹：

- `originator: pi`
- `User-Agent: pi (...)`
- `User-Agent: pi-coding-agent`
- `x-opencode-client: pi`
- OpenRouter：`HTTP-Referer: https://pi.dev`
- OpenRouter：`X-OpenRouter-Title: pi`
- OpenRouter：`X-OpenRouter-Categories: cli-agent`
- NVIDIA：`X-BILLING-INVOKE-ORIGIN: Pi`

安全改写建议：

- 可改写：`originator`、`User-Agent`、`x-opencode-client`、OpenRouter/NVIDIA 归因头
- 不应全局替换：`pi` 单词、系统提示内容、文件路径、包名、工具名映射、Anthropic OAuth 认证逻辑

## Kimi Code 0.35.0

注意：本机源码参考主要是 `/home/ubuntu/temp/kimi-code-0.33.0`，当前安装是 `0.35.0`，核心结构应一致，但 0.34.0–0.35.0 的增量未逐一验证。

system prompt 注入方式：

- OpenAI/Kimi Chat：`messages.push({ role: 'system', content: systemPrompt })`
- OpenAI Responses：`createParams['instructions'] = systemPrompt`
- Gemini：`systemInstruction`
- Anthropic：顶层 `system`

内置工具约 27 个：

- `Read`
- `Glob`
- `Write`
- `Grep`
- `Bash`
- `Edit`
- `Agent`
- `AgentSwarm`
- `FetchURL`
- `AskUserQuestion`
- `WebSearch`
- `TodoList`
- `EnterPlanMode`
- `ExitPlanMode`
- `Skill`
- `TaskList`
- `TaskOutput`
- `TaskStop`
- `CreateGoal`
- `GetGoal`
- `UpdateGoal`
- `SetGoalBudget`
- `ReadMediaFile`
- `SelectTools`
- `CronCreate`
- `CronDelete`
- `CronList`

请求头指纹：

- `User-Agent: kimi-code-cli/<version>`
- `X-Msh-Platform: kimi_code_cli`
- `X-Msh-Version`
- `X-Msh-Device-Name`
- `X-Msh-Device-Model`
- `X-Msh-Os-Version`
- `X-Msh-Device-Id`

安全改写建议：

- 可改值：`User-Agent`、`X-Msh-Platform`
- 不改名：`X-Msh-*` 头名是上游识别协议的一部分
- 不改：`Authorization`、本地 OAuth storage key、npm 包名、工具名

## Codex CLI 0.147.0

安装路径：`/home/ubuntu/.nvm/versions/node/v24.18.0/lib/node_modules/@openai/codex`

system prompt 注入方式：

- 倾向使用 Responses API 顶层 `instructions`
- 源码二进制中可见 `systemPrompt`、`base_instructions`、`developer_instructions` 等

内置工具约 16 类：

- `shell_tool` / `unified_exec`
- `apply_patch`
- `view_image`
- `web_search`
- `update_plan`
- `spawn_agent`
- `sleep`
- `get_context_remaining`
- `wait_for_environment`
- `request_user_input`
- `request_permissions`
- `read_mcp_resource`
- `list_mcp_resources`
- `image_gen` / `image_generation`
- `js_repl`
- `exec`

请求头指纹：

- `originator: codex_cli_rs`
- `originator: codex-tui`
- `originator: codex_vscode`
- `User-Agent: codex-cli/0.147.0`
- `ChatGPT-Account-Id`
- `OAI-Product-Sku`
- `x-openai-internal-codex-residency`
- `x-codex-installation-id`
- `x-oai-attestation`
- `x-openai-actor-authorization`
- `X-OpenAI-Fedramp`
- `x-client-request-id`
- `OpenAI-Beta`
- `x-codex-turn-state`

安全改写建议：

- 可改值：`originator`、`User-Agent`
- 不改：`ChatGPT-Account-Id`、`OAI-Product-Sku`、`OpenAI-Beta`、`Authorization`、安全证明头、工具名、模型名

## Gemini CLI 0.53.1

安装路径：`/home/ubuntu/.nvm/versions/node/v24.18.0/lib/node_modules/@google/gemini-cli`

system prompt 注入方式：

- 顶层 `systemInstruction`
- 官方 API：`systemInstruction: { role, parts: [{ text }] }`

内置工具至少 10 个：

- `glob`
- `grep_search`
- `list_directory`
- `read_file`
- `run_shell_command`
- `write_file`
- `replace`
- `google_web_search`
- `web_fetch`
- `read_many_files`

另有可选/扩展：

- `write_todos`
- `get_internal_docs`
- `activate_skill`

请求头指纹：

- `User-Agent: Google-Gemini-CLI/...`
- `x-goog-api-client: gl-node/<nodeVersion>`
- `x-goog-api-client: gdcl/<pkg.version> gl-node/<nodeVersion>`
- `X-Goog-Api-Key`
- `X-Goog-Signature`

安全改写建议：

- 可改值：`User-Agent`、`x-goog-api-client`
- 不改名：`X-Goog-*` 头名
- 不改：`X-Goog-Api-Key`、模型名、工具名

## Claude Code 2.1.226

安装路径：`/home/ubuntu/.local/share/claude/versions/2.1.226`

system prompt 注入方式：

- Anthropic Messages 顶层 `system`
- 新版也支持 mid-conversation system block，但主路径仍是顶层 `system`

内置工具约 18 个：

- `Bash`
- `Read`
- `Write`
- `Edit`
- `MultiEdit`
- `Glob`
- `Grep`
- `Task`
- `TaskList`
- `TaskOutput`
- `TaskStop`
- `BashOutput`
- `KillShell`
- `WebSearch`
- `WebFetch`
- `TodoWrite`
- `NotebookEdit`
- `ListFiles`

请求头指纹：

- `User-Agent: claude-code/<version>`
- `anthropic-client-platform: claude_code_cli` / `claude_code_vscode` / 等
- `anthropic-version: 2023-06-01`
- `anthropic-beta: ...`
- `x-stainless-*`
- `x-client-request-id`
- `x-app-name`
- `x-app-ver`
- `AI_AGENT: claude-code_2_1_226_harness` 类环境标识

安全改写建议：

- 可改值：`User-Agent`、`anthropic-client-platform`、`x-app-name`、`x-app-ver`
- 不改：`anthropic-version`、`anthropic-beta` 功能开关、`x-stainless-*` 头名、文件路径、本地配置名、工具名、模型名

## 当前插件缺口

当前 `custom_mappings` 不能完整覆盖：

1. Kimi 的 `messages[0].role = system`
2. Kimi/Codex/Pi 的 Responses `instructions`
3. Gemini/AGY 的 `systemInstruction`
4. 各客户端 HTTP 请求头
5. Gemini/AGY 工具声明风格
6. Pi 的 provider 分散 system 注入

## 推荐插件升级设计

原则：只改客户端身份标识，不改用户消息、模型参数、工具协议。

推荐扩展字段：

- 保留：JSON 键名为 `system`
- 新增：`systemInstruction`
- 新增：`instructions`
- 新增：`messages[]` 中 `role == "system"` 或 `role == "developer"` 的 `content`

推荐请求头改写白名单：

- `User-Agent`
- `originator`
- `x-opencode-client`
- `X-Msh-Platform`
- `anthropic-client-platform`
- `x-goog-api-client`

明确不改：

- `Authorization`
- `anthropic-version`
- `anthropic-beta`
- `ChatGPT-Account-Id`
- `OAI-Product-Sku`
- `OpenAI-Beta`
- `X-Goog-Api-Key`
- `tools[]`
- 模型名
- 用户消息内容

推荐 `custom_mappings`：

```yaml
custom_mappings:
  Kimi Code: Antigravity
  Kimi: Antigravity
  Codex: Antigravity
  Claude Code: Antigravity
  Gemini CLI: Antigravity
  pi-coding-agent: Antigravity
```

不推荐全局使用短词：

- `pi`
- `agy`
- `kimi`

这些短词应在请求头白名单或明确身份字段中处理，不应在任意 JSON 文本中全局替换。

## 结论

- 有必要调研 `agy`：它的 system 结构、工具名集合和 header 指纹决定 Antigravity 身份是否可信。
- 当前插件仅靠 `system` 字段关键词匹配，覆盖不完整。
- 改写 header 不需要修改 CPA，只需要修改插件。
- 若要防封，合理方案是升级插件为“客户端身份重写器”。
- 但即使完整改写 body/header，也不能保证绝对防封：上游还可能检查 TLS/JA3、session 状态、payload 元数据、请求 ID 格式、配额路径等。

## 附录：2026-08-12 antigravity/gemini-cli 全链路 403 事故复盘

### 症状

- CPA 所有 antigravity 模型报 `400 antigravity auth missing project_id: no project_id in response`（onboarding 取不到 project_id）。
- 随后 gemini-cli 与 antigravity 两条凭证均报 `403 CONSUMER_INVALID: Permission denied on resource project gen-lang-client-<redacted>`（service: cloudaicompanion.googleapis.com）。
- 同一 Google 账号在 Cloud Shell 里的 gemini 正常（Cloud Shell 走 Google 托管的 cloud-shell 项目，不依赖用户项目）。

### 根因

`gcloud projects describe gen-lang-client-<redacted>` 显示 `lifecycleState: DELETE_REQUESTED`——项目正在被 Google 删除（此前因合规被停用，恢复邮件不等于撤销删除流程）。项目一删，绑定它的两条 OAuth 凭证全部 403。

### 修复

1. 新建项目 `<replacement-gcp-project>`，启用 `cloudaicompanion.googleapis.com`。
2. 将 CPA 两条凭证文件（`antigravity-*.json`、`*-gemini-cli.json`）的 `project_id`（及 `project_ids`）改为 `<replacement-gcp-project>`。OAuth token 是账号级、与项目无关，无需重新授权。
3. 两条链路恢复 200；pi 实测通过。

### 经验

- CPA antigravity executor 的 project_id 可直接写进凭证文件跳过 onboarding；字段在 token 刷新重写后仍保留。
- 修改 bind-mount 凭证文件必须原子写（同目录 tmp + mv），否则 watcher 读到截断空文件会产生幽灵客户端；幽灵客户端刷新 token 时会把临时文件名写回磁盘，需删文件才能清除。
- `gen-lang-client-*` 是 AI Studio 自动创建的项目，合规停用/删除会同时打死 gemini-cli 与 antigravity 两条链路。
- 旧项目 `gen-lang-client-<redacted>` 已用 `gcloud projects undelete` 恢复为 ACTIVE，但其 serviceusage 资源未完全恢复（启用服务报 RESOURCES_NOT_FOUND），故弃用，改用新项目。

## 2026-08-12 CPA / GPT-5.5 模型元数据审计

### 审计口径与 alias 边界

- 本节只核对模型元数据，不评价模型能力，也不推断 CPA/Kimi 渠道的实际路由。
- 表内数值只适用于“配置名与厂商官方模型 ID 完全一致”的情况。Anthropic 从 4.6 起采用无日期但仍固定的 model ID；Google 的 stable/preview ID 和 OpenAI 的标准 `gpt-5.5` 也必须逐字匹配其官方页面。
- “Context / input limit”保留厂商原始口径：Anthropic/OpenAI 发布 context window，Google 发布 input token limit；两者不做加减换算。
- 任何带渠道前后缀或额外限额标记的名称，例如含 `100k`、`codex/`、`cx/` 的名称，均是渠道自定义 alias，不是这里核对的官方 ID。除非该渠道另有一手文档或接口返回证明，否则其 context/input limit、max output、image input、tool use、thinking/reasoning 选项及能否关闭全部记为**未确认**，不得套用对应基础模型规格。
- 上述 alias 仅用于说明判定规则，不表示本仓库当前一定存在这些名称。

### 官方基础模型规格

| 配置名 | 官方身份与状态 | Context / input limit | Max output | Image input | Tool use | Thinking / reasoning 控制 | 可完全关闭 |
| --- | --- | ---: | ---: | --- | --- | --- | --- |
| `claude-sonnet-4-6` | Anthropic 官方固定 snapshot；无日期后缀也不是滚动 alias | 1M tokens | 128k tokens（同步 Messages API；见下方文档冲突说明） | 支持 | 支持 | `effort`: `low` / `medium` / `high`（默认）/ `max`；不支持 `xhigh`。推荐配合 `thinking: {type: "adaptive"}`；手动 `enabled + budget_tokens` 仍可用但已 deprecated | 是；不传 `thinking` 即不启用 thinking，也可显式 `disabled`。`effort` 本身不是开关 |
| `gemini-2.5-pro` | Google 官方 stable model | 1,048,576 input tokens | 65,536 tokens | 支持 | 支持（Function calling） | GenerateContent: `thinkingBudget` 为 `128..32768`，`-1` 为动态且默认；Interactions: `thinking_level` 为 `low` / `medium` / `high`，默认 On | 否 |
| `gemini-2.5-flash` | Google 官方 stable model | 1,048,576 input tokens | 65,536 tokens | 支持 | 支持（Function calling） | GenerateContent: `thinkingBudget` 为 `0..24576`，`-1` 为动态且默认；Interactions: `thinking_level` 为 `low` / `medium` / `high`，默认 On | 是；GenerateContent 设 `thinkingBudget: 0` |
| `gemini-2.5-flash-lite` | Google 官方 stable model | 1,048,576 input tokens | 65,536 tokens | 支持 | 支持（Function calling） | GenerateContent: 默认不思考，启用预算 `512..24576`，`0` 关闭、`-1` 动态；Interactions: `thinking_level` 为 `low` / `medium` / `high`，默认 Off | 是；默认 Off，或 GenerateContent 设 `thinkingBudget: 0` |
| `gemini-3.1-flash-lite` | Google 官方 stable/GA model | 1,048,576 input tokens | 65,536 tokens | 支持 | 支持（Function calling） | `thinking_level`: `minimal`（默认）/ `low` / `medium` / `high` | 否；`minimal` 对多数请求近似 no thinking，但官方明确不保证完全关闭 |
| `gemini-3.1-flash-lite-preview` | 曾是 Google 官方 preview endpoint；已于 2026-05-25 shut down | 1,048,576 input tokens | 65,536 tokens | 支持 | 支持（Function calling） | 官方归档页确认支持 Thinking，但当前一手文档未继续单列该已停服 endpoint 的可选档位：**未确认** | **未确认** |
| `gpt-5.5` | OpenAI 官方标准 model ID/alias；不含任何渠道前缀 | 1,050,000 tokens | 128,000 tokens | 支持（input only） | 支持：Function calling；Responses API 另支持 Web/File search、Image generation、Code interpreter、Hosted shell、Apply patch、Skills、Computer use、MCP、Tool search | `reasoning.effort`: `none` / `low` / `medium`（默认）/ `high` / `xhigh`；该型号页未列 `minimal` 或 `max` | 是；`reasoning.effort: "none"` |

补充说明：

- Anthropic 当前 Models overview 与 Context windows 页面将 `claude-sonnet-4-6` 的单请求上限列为 128k；`What's new in Claude 4.6` 页面仍残留“Sonnet 4.6 为 64k”的发布时描述，属于官方一手文档冲突。本表采用当前模型总表的 128k；如该值用于生产侧硬限制，应再以 Models API 返回的 `max_tokens` 或实际账户请求验证。Message Batches API 另可在携带 `output-300k-2026-03-24` beta header 时到 300k，不能当作同步 Messages API 默认值。
- Google 有两套不同接口控制面：GenerateContent 使用数值型 `thinkingBudget`，Interactions 使用枚举型 `thinking_level`；它们不是同一个参数，不能混写。
- “支持 Thinking”只证明模型具备思考能力，不自动证明某个档位或关闭方式可用。因此已停服 preview endpoint 的档位和关闭能力保持“未确认”。
- `minimal` 与完全关闭不同：Google 明确写明 Gemini 3.x 的 `minimal` 不保证 thinking 为零。

### CPA v7.2.128 的实际代理规格

上表是厂商官方基础模型规格，不应直接覆盖到 CPA。当前 CPA `/v1/models` 返回的 `owned_by` 表明，这六个名称实际分成两条路由：

- `claude-sonnet-4-6`、`gemini-3.1-flash-lite`：`antigravity`
- `gemini-2.5-pro`、`gemini-2.5-flash`、`gemini-2.5-flash-lite`、`gemini-3.1-flash-lite-preview`：Google / `gemini-cli`

本机镜像是 CLIProxyAPI `v7.2.128`、commit `bd34cec`。该 commit 的内置 registry 和 OpenAI translator 给出以下实际代理口径：

| CPA 模型 | 实际路由 | CPA context / input | CPA max output | Kimi 可表达的有效 effort | Thinking Off |
| --- | --- | ---: | ---: | --- | --- |
| `claude-sonnet-4-6` | Antigravity | 200,000 | 64,000 | `low` / `medium` / `high` / `xhigh` / `max`；`minimal` 会钳制成 `low` | `none` 可关闭 |
| `gemini-2.5-pro` | gemini-cli | 1,048,576 | 65,536 | `minimal` / `low` / `medium` / `high` / `xhigh`；`max` 会钳制成 `xhigh` | 不可关闭；`none` 会钳制到最小预算 128 |
| `gemini-2.5-flash` | gemini-cli | 1,048,576 | 65,536 | `minimal` / `low` / `medium` / `high`；`xhigh` / `max` 会钳制成 `high` | `none` 可关闭 |
| `gemini-2.5-flash-lite` | gemini-cli | 1,048,576 | 65,536 | `minimal` / `low` / `medium` / `high`；`xhigh` / `max` 会钳制成 `high` | `none` 可关闭 |
| `gemini-3.1-flash-lite` | Antigravity | 1,048,576 | 65,535 | `minimal` / `low` / `medium` / `high` | `none` 可关闭（CPA registry 明确允许 zero） |
| `gemini-3.1-flash-lite-preview` | gemini-cli | 1,048,576 | 65,536 | `minimal` / `low` / `medium` / `high` | 官方 3.1 模型不能完全关闭；省略配置会回到上游默认 |

CLIProxyAPI 会先把 OpenAI `reasoning_effort` 映射为 thinking level，再按目标 provider 的 registry 转成 level 或 budget 并钳制。因此，Kimi 的 `support_efforts` 应列出上表中能产生不同实际效果的值，不应把会被钳制成同一档的值重复暴露。`none` 只应出现在确实能关闭 thinking 的模型上。

CPA 的 OpenAI translator 会保留 function tools，并把 `image_url`、`video_url`、`input_audio` 转成 Gemini/Antigravity 的内联多模态 part。但 Kimi 的 `openai` provider 对用户音频生成的是非标准 `audio_url`，而 CPA translator 这里只处理 `input_audio`，所以当前组合只能可靠声明 `tool_use`、`image_in`、`video_in` 和 `thinking`，不能声明 `audio_in`。Claude 这条 Antigravity 路由只应声明 Claude 实际支持的 text/image、tool use 和 thinking，不能因 translator 能解析音视频就虚报模型能力。

一手来源：

- CPA registry（commit `bd34cec`）：https://github.com/router-for-me/CLIProxyAPI/blob/bd34cec/internal/registry/models/models.json
- OpenAI → Antigravity translator：https://github.com/router-for-me/CLIProxyAPI/blob/bd34cec/internal/translator/antigravity/openai/chat-completions/antigravity_openai_request.go
- OpenAI → Gemini translator：https://github.com/router-for-me/CLIProxyAPI/blob/bd34cec/internal/translator/gemini/openai/chat-completions/gemini_openai_request.go
- level ↔ budget 映射：https://github.com/router-for-me/CLIProxyAPI/blob/bd34cec/internal/thinking/convert.go

注意：`gemini-3.1-flash-lite-preview` 的 Google 官方 endpoint 已停服，但 CPA 仍将它列在 `/v1/models`。目录存在不等于端到端可调用；在不做真实 smoke test 的前提下，只能标记为“CPA 已注册、实际可用性未确认”，不能直接删除，也不能宣称仍可用。

### 逐项官方来源

- `claude-sonnet-4-6`
  - Models overview（模型 ID、固定 snapshot、context、max output、image input）：https://platform.claude.com/docs/en/about-claude/models/overview
  - Context windows（当前 1M/128k 口径）：https://platform.claude.com/docs/en/build-with-claude/context-windows
  - Claude 4.6 发布页（64k 冲突项、adaptive/manual thinking）：https://platform.claude.com/docs/en/about-claude/models/whats-new-claude-4-6
  - Effort（可选值、默认值、`xhigh` 边界）：https://platform.claude.com/docs/en/build-with-claude/effort
  - Adaptive thinking / thinking 开关：https://platform.claude.com/docs/en/build-with-claude/adaptive-thinking
  - Tool use：https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview
- `gemini-2.5-pro`
  - 型号页：https://ai.google.dev/gemini-api/docs/models/gemini-2.5-pro
  - GenerateContent thinking：https://ai.google.dev/gemini-api/docs/generate-content/thinking
  - Interactions thinking：https://ai.google.dev/gemini-api/docs/thinking
- `gemini-2.5-flash`
  - 型号页：https://ai.google.dev/gemini-api/docs/models/gemini-2.5-flash
  - GenerateContent thinking：https://ai.google.dev/gemini-api/docs/generate-content/thinking
  - Interactions thinking：https://ai.google.dev/gemini-api/docs/thinking
- `gemini-2.5-flash-lite`
  - 型号页：https://ai.google.dev/gemini-api/docs/models/gemini-2.5-flash-lite
  - GenerateContent thinking：https://ai.google.dev/gemini-api/docs/generate-content/thinking
  - Interactions thinking：https://ai.google.dev/gemini-api/docs/thinking
- `gemini-3.1-flash-lite`
  - 型号页：https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-lite
  - Thinking levels 与 `minimal` 边界：https://ai.google.dev/gemini-api/docs/generate-content/thinking
- `gemini-3.1-flash-lite-preview`
  - 归档型号页：https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-lite-preview
  - Changelog（2026-05-25 shut down）：https://ai.google.dev/gemini-api/docs/changelog
- `gpt-5.5`
  - 型号页（context、max output、modalities、tools、该型号 effort 精确集合）：https://developers.openai.com/api/docs/models/gpt-5.5
  - Reasoning guide（`none` 语义、reasoning token 口径）：https://developers.openai.com/api/docs/guides/reasoning

### 审计结论

- 六个 CPA/Kimi 名称均是当前或历史上真实存在的 Google/Anthropic 官方 exact ID；其中五个当前仍可用，`gemini-3.1-flash-lite-preview` 已停服，不能再当作可用模型。
- 标准 OpenAI `gpt-5.5` 的规格只能用于精确的官方名称 `gpt-5.5`；`codex/`、`cx/` 或其他渠道前缀版本仍是未确认的代理 alias。
- 官方基础模型的上限与能力不等于 CPA/Kimi 代理层的实际值；代理可能进一步裁剪 context/output、模态、工具或推理档位，必须以渠道一手证据或端到端实测为准。
