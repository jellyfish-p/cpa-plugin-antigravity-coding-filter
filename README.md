# Antigravity Coding Filter

CLIProxyAPI v7 dynamic plugin for blocking or rewriting non-Antigravity coding software signals.

## Filter Modes

The plugin inspects only system/developer prompt text after CLIProxyAPI has selected the upstream format. Choose how matches are handled with `mode`:

- `block` (default): reject the entire request with HTTP `403 Forbidden` and a `blocked_by_antigravity_coding_filter` error.
- `rewrite`: replace matched names with `Antigravity` and forward the request.

Matching is case-insensitive. The eligible locations are:

- Anthropic `system` text and text blocks.
- OpenAI Responses `instructions`, plus `input` messages whose role is `system` or `developer`, including `input_text` blocks.
- OpenAI Chat Completions `messages` whose role is `system` or `developer`, including string and `text` block content.
- Gemini `systemInstruction.parts[].text` and `system_instruction.parts[].text`, including the Antigravity `request` wrapper.

User and assistant messages, tool definitions/results, image data, metadata, and other JSON fields are preserved. Technical references such as URLs, paths, `CODEX.md`, `.codex`, and `mcp__...` identifiers are preserved even when they occur inside an eligible prompt. Markdown emphasis around a name does not prevent a configured name from being rewritten.

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

## Mapping Configuration

You can select rewrite mode, disable the built-in preset, and provide your own mapping relationships in the plugin config:

```yaml
plugins:
  configs:
    antigravity-coding-filter:
      enabled: true
      priority: 1
      mode: rewrite
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
