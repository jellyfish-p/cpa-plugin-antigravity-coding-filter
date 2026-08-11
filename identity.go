package main

// identity.go extends the base filter from "system-field only" into a
// client-identity rewriter: it covers the remaining identity-bearing body
// fields used by pi / Kimi Code / Codex CLI / Gemini CLI / Claude Code and
// rewrites a small whitelist of client-identity request headers.
//
// Design rules (see docs/research.md):
//   - Only identity fields are rewritten: top-level "system",
//     "systemInstruction", "instructions", and the "content" of
//     messages[]/input[] entries whose role is "system" or "developer".
//   - User messages, tools, model names and auth material are never touched.
//   - Bare short words (e.g. "pi") are only allowed in header_mappings,
//     never in body mappings.

import (
	"net/http"
	"strings"
)

// identityHeaderNames is the whitelist of request headers whose VALUES may be
// rewritten (header names themselves are never changed). Everything else —
// Authorization, anthropic-version, anthropic-beta, chatgpt-account-id,
// oai-product-sku, openai-beta, x-goog-api-key, x-stainless-*, the remaining
// x-msh-* headers — is left untouched.
var identityHeaderNames = map[string]struct{}{
	"user-agent":                {},
	"originator":                {},
	"x-opencode-client":         {},
	"x-msh-platform":            {},
	"anthropic-client-platform": {},
	"x-goog-api-client":         {},
}

// isIdentityFieldKey reports whether a JSON object key carries the client
// system prompt / identity text for one of the supported API shapes:
//   - "system": Anthropic Messages, Bedrock, chat-completions style bodies.
//   - "systemInstruction": Gemini generateContent.
//   - "instructions": OpenAI Responses.
func isIdentityFieldKey(key string) bool {
	return key == "system" || key == "systemInstruction" || key == "instructions"
}

// isRoleMessageListKey reports whether a JSON object key holds a message list
// whose entries may carry a role-typed system/developer message:
//   - "messages": chat completions / Anthropic.
//   - "input": OpenAI Responses.
func isRoleMessageListKey(key string) bool {
	return key == "messages" || key == "input"
}

// rewriteIdentityFields walks the decoded request body and rewrites identity
// text in place, returning whether anything changed. It replaces the direct
// rewriteSystemFields entry point so that all supported identity fields are
// covered by a single walk.
func rewriteIdentityFields(value any, mappings []rewriteMapping) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		changed := false
		for key, child := range typed {
			if isIdentityFieldKey(key) {
				next, childChanged := rewriteSystemValue(child, mappings)
				if childChanged {
					typed[key] = next
					changed = true
				}
				continue
			}
			if isRoleMessageListKey(key) {
				if next, childChanged := rewriteRoleMessages(child, mappings); childChanged {
					typed[key] = next
					changed = true
					continue
				}
			}
			next, childChanged := rewriteIdentityFields(child, mappings)
			if childChanged {
				typed[key] = next
				changed = true
			}
		}
		return typed, changed
	case []any:
		changed := false
		for i, child := range typed {
			next, childChanged := rewriteIdentityFields(child, mappings)
			if childChanged {
				typed[i] = next
				changed = true
			}
		}
		return typed, changed
	default:
		return value, false
	}
}

// rewriteRoleMessages rewrites the "content" of message entries whose role is
// "system" or "developer". Entries with any other role (user, assistant,
// tool, ...) are passed through untouched. Non-list values (e.g. a plain
// string "input" in the Responses API) are left alone.
func rewriteRoleMessages(value any, mappings []rewriteMapping) (any, bool) {
	items, ok := value.([]any)
	if !ok {
		return value, false
	}
	changed := false
	for i, item := range items {
		message, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role, _ := message["role"].(string)
		if role != "system" && role != "developer" {
			continue
		}
		content, exists := message["content"]
		if !exists {
			continue
		}
		next, childChanged := rewriteSystemValue(content, mappings)
		if childChanged {
			message["content"] = next
			items[i] = message
			changed = true
		}
	}
	return items, changed
}

// effectiveHeaderMappings merges the body mappings (defaults + custom) with
// the header-only mappings and normalizes the result (longest match first).
// Header-only mappings exist so that UA/originator spellings such as
// "kimi-code-cli", "codex_cli_rs" or the bare word "pi" can be matched
// without ever applying those short tokens to request bodies.
func effectiveHeaderMappings(cfg filterConfig) []rewriteMapping {
	mappings := append(effectiveMappings(cfg), cfg.HeaderMappings...)
	return normalizeMappings(mappings)
}

// rewriteIdentityHeaders rewrites the values of whitelisted identity headers
// and returns only the headers that actually changed. The response merge
// semantics ("replaces matching current request headers and preserves headers
// not mentioned here") mean returning just the changed subset is enough.
func rewriteIdentityHeaders(headers http.Header, mappings []rewriteMapping) (http.Header, bool) {
	if len(headers) == 0 {
		return nil, false
	}
	out := make(http.Header)
	for name, values := range headers {
		if _, ok := identityHeaderNames[strings.ToLower(name)]; !ok {
			continue
		}
		rewritten := make([]string, len(values))
		valueChanged := false
		for i, value := range values {
			next := value
			for _, mapping := range mappings {
				var replaced bool
				next, replaced = replaceInsensitive(next, mapping.Match, mapping.Replacement)
				valueChanged = valueChanged || replaced
			}
			rewritten[i] = next
		}
		if valueChanged {
			out[name] = rewritten
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}
