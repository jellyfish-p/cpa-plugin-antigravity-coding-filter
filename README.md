# Antigravity Coding Filter

CLIProxyAPI v7 dynamic plugin for rewriting non-Antigravity coding software signals to Antigravity.

## Rewrite Rules

The plugin rewrites configured coding-client names when they appear inside JSON fields named `system`.

The built-in mapping preset is enabled by default:

- `OpenCode` -> `Antigravity`
- `Codex` -> `Antigravity`
- `Claude Code` -> `Antigravity`

Matching is case-insensitive and only scans `system`. Mentions in user prompts, `messages`, or other fields are not rewritten.

## Mapping Configuration

You can disable the built-in preset and provide your own mapping relationships in the plugin config:

```yaml
plugins:
  configs:
    antigravity-coding-filter:
      enabled: true
      priority: 1
      use_default_keywords: false
      custom_mappings:
        Cursor: Antigravity
        Windsurf: Antigravity
        JetBrains AI: Antigravity
```

`custom_mappings` also accepts a comma- or newline-delimited `from: to` string for simpler one-line config. Blank entries and duplicate source names are ignored.

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
