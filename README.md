# Antigravity Coding Filter

CLIProxyAPI v7 dynamic plugin for blocking or rewriting non-Antigravity coding software signals.

## Filter Modes

The plugin inspects requests after CLIProxyAPI has selected the upstream format. The `mode` setting controls how the plugin handles matches.

- `block` (default): reject the entire request with HTTP `403 Forbidden` and a `blocked_by_antigravity_coding_filter` error.
- `rewrite`: replace matched names in the selected text blocks and forward the request.

Matching is case-insensitive. Both modes support these system/developer prompt locations.

- Anthropic `system` text and text blocks.
- OpenAI Responses `instructions`, plus `input` messages whose role is `system` or `developer`, including `input_text` blocks.
- OpenAI Chat Completions `messages` whose role is `system` or `developer`, including string and `text` block content.
- Gemini `systemInstruction.parts[].text` and `system_instruction.parts[].text`, including the Antigravity `request` wrapper.

Technical references such as URLs, paths, `CODEX.md`, `.codex`, and `mcp__...` identifiers are preserved within eligible text. Markdown emphasis around a configured name still allows a match.

The before-auth interceptor and model router are pass-through hooks. The filter runs in `request.intercept_after` only when CLIProxyAPI reports `ToFormat: "antigravity"`; other upstream providers are untouched.

HTTP 403 propagation for after-auth blocking requires CLIProxyAPI v7.3.2 or newer, which provides interceptor termination fields. Earlier hosts do not understand the plugin RPC termination response.

## Built-in Keywords

The built-in preset is enabled by default and covers major AI coding editors, assistants, terminal agents, and related general-purpose agents:

- Claude Agent, Claude Code, OpenAI Codex / Codex CLI, OpenCode
- GitHub Copilot / Copilot CLI, Gemini Code Assist / Gemini CLI
- Cursor, Windsurf / Codeium, Cline, Roo Code, Kilo Code, Aider, Continue.dev
- Amazon Q Developer / CodeWhisperer, JetBrains AI Assistant / Junie, Kiro
- Qoder / Qoder CLI, Qwen Code, Trae, Tabnine, Sourcegraph Cody, Augment Code
- Replit Agent / Ghostwriter, Devin, OpenHands, SWE-agent, Goose
- Zed AI, Void Editor, PearAI, Refact.ai, Tabby, GitLab Duo, Visual Studio IntelliCode
- CodeBuddy, Blackbox AI, Pieces for Developers, Qodo / CodiumAI, Rovo Dev CLI, Factory Droid
- OpenClaw (including Clawdbot and Moltbot), Hermes Agent, WorkBuddy
- DeepSeek Harness (`dsh`) and Pi coding agent, using identity phrases from their default system prompts. [DeepSeek Harness prompt snapshot](https://github.com/deepseek-ai/deepseek-harness/blob/2e39e70496aa02986ca6a820c5d497e816b69d7e/snapshots/sdk/text-turn/system-prompt.expected.md), [Pi prompt builder](https://github.com/earendil-works/pi/blob/63dec2621d3cb2e1f4d7cefd6df061e78c6de979/packages/coding-agent/src/core/system-prompt.ts)

## Scope

Set `scope` to any combination of the text blocks below. Both `block` and `rewrite` check the selected blocks. Omit `scope` to use `[system]` or set it to `[]` to disable matching.

| Option | Text selected |
| --- | --- |
| `system` | System and developer prompts |
| `user` | User message text and OpenAI Responses string `input` |
| `assistant` | Assistant or Gemini `model` message text |
| `tool_results` | Tool and function message text, Anthropic `tool_result` content, OpenAI Responses output items, and Gemini function result text |
| `tool_descriptions` | `description` in Anthropic and OpenAI Responses tool definitions, Chat Completions function definitions, and Gemini function declarations |

Text carried in a plain Gemini user part is covered by `user`. The `tool_results` option covers text in explicitly structured function responses or tool messages. A tool result delivered as a string is processed as text, including a string containing serialized JSON. The `tool_descriptions` option applies the configured mappings to each tool's own description. Tool names, parameter schemas, protocol-level call identifiers, function arguments, images, metadata, and structured result fields outside the supported text locations are preserved.

The CLIProxyAPI Management Center edits array settings as JSON. Enter a list such as `["system", "tool_descriptions"]` in the `scope` field, or set the same options in YAML. Existing `rewrite_scope` settings still apply in `rewrite` mode when `scope` is omitted.

## Mapping Configuration

The plugin config can select a mode and scope, disable the built-in preset, and provide custom mappings.

```yaml
plugins:
  configs:
    antigravity-coding-filter:
      enabled: true
      priority: 1
      mode: rewrite
      scope: [system, user, tool_results, tool_descriptions]
      use_default_keywords: false
      custom_mappings:
        Cursor: Antigravity
        Windsurf: Antigravity
        JetBrains AI: Antigravity
```

In `block` mode, each `custom_mappings` key is an additional blocked keyword and its value is ignored. In `rewrite` mode, the key is replaced with its value. The field also accepts a comma- or newline-delimited `from: to` string. Blank entries and duplicate source names are ignored.

## Build

CLIProxyAPI dynamic plugins require CGO. Confirm `CGO_ENABLED=1` before building.

Windows amd64:

```powershell
go build -buildmode=c-shared -o plugins/windows/amd64/antigravity-coding-filter.dll .
Remove-Item plugins/windows/amd64/antigravity-coding-filter.h
```

The plugin ID is derived from the dynamic library filename, so this build path registers the plugin as `antigravity-coding-filter`.

## CLIProxyAPI Config

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    antigravity-coding-filter:
      enabled: true
      priority: 1
      mode: block
      scope: [system]
      use_default_keywords: true
      custom_mappings: {}
```

CLIProxyAPI searches `plugins/<GOOS>/<GOARCH>-<variant>`, then `plugins/<GOOS>/<GOARCH>`, then `plugins`.

## Runtime Verification

After starting CLIProxyAPI, call:

```text
GET /v0/management/plugins
```

Confirm the plugin reports `registered: true` and `effective_enabled: true`.

## Tests

```powershell
go test ./...
```
