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
- `kimi 0.34.0`
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

## Kimi Code 0.34.0

注意：本机源码参考主要是 `/home/ubuntu/temp/kimi-code-0.33.0`，当前安装是 `0.34.0`，核心结构应一致，但 0.34.0 的增量未逐一验证。

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
