# Antigravity Coding Filter

CLIProxyAPI v7 dynamic plugin for protecting the Antigravity route from non-Antigravity coding software traffic.

## Detection Rules

The plugin blocks a request when its JSON body contains any of these signals:

- `system` contains one of: `OpenCode`, `Codex`, `Claude Code`
- any JSON object contains `prompt_cache_key`
- any JSON object contains `metadata.user_id`

Keyword matching is case-insensitive and only scans `system`. Mentions in user prompts, `messages`, or other fields do not block by themselves.

## Build

Windows amd64:

```powershell
go build -buildmode=c-shared -o plugins/windows/amd64/antigravity-coding-filter.dll .
Remove-Item plugins/windows/amd64/antigravity-coding-filter.h
```

## CLIProxyAPI Config

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    antigravity-coding-filter:
      enabled: true
      priority: 1
```

## Tests

```powershell
go test ./...
```
