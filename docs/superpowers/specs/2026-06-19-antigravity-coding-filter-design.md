# Antigravity Coding Filter Design

## Goal

Build a CLIProxyAPI dynamic plugin that protects the Antigravity route by blocking requests that look like non-Antigravity coding software traffic.

## Scope

The plugin uses CLIProxyAPI v7's dynamic plugin RPC contract. It declares `model_router` and `executor` capabilities. The router inspects requests before the normal provider/auth path. If the request appears to be a non-Antigravity coding software request, the router sends it to the plugin's own executor. The executor returns a JSON block response. Otherwise the router returns `Handled:false` so CLIProxyAPI continues its native routing.

Only unit tests are required. Tests cover the request classifier, route decision, and block response payload without launching CLIProxyAPI or loading a compiled dynamic library.

## Detection

The classifier inspects JSON request bodies only.

It blocks when any of these signals are present:

- A `system` field contains a configured coding-client keyword. Defaults: `OpenCode`, `Codex`, `Claude Code`.
- A `prompt_cache_key` field exists anywhere in the JSON object tree.
- A `metadata.user_id` path exists anywhere in the JSON object tree.

Keyword matching is case-insensitive. Keywords are only checked inside `system`; the same words in user prompts, messages, or other fields do not block by themselves.

## Plugin Methods

- `plugin.register` and `plugin.reconfigure` return schema version 1, metadata, and capabilities.
- `model.route` decodes the request and returns `Handled:true`, `TargetKind:"self"` when classification blocks it.
- `executor.execute` and `executor.execute_stream` return a block response.
- Unsupported methods return an error envelope.

## Configuration

The first implementation has no runtime configuration parsing. It always uses the default keyword set and structural signals listed above.

## Testing

Unit tests verify:

- System keyword matches block.
- Keywords outside `system` do not block.
- `prompt_cache_key` existence blocks.
- `metadata.user_id` existence blocks.
- Clean requests pass through.
- Route and executor methods encode the expected plugin decisions.
