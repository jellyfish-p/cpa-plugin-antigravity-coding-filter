package main

/*
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef int (*cliproxy_host_call_fn)(void*, char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_host_free_fn)(void*, size_t);

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	cliproxy_host_call_fn call;
	cliproxy_host_free_fn free_buffer;
} cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cliproxy_plugin_call(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxy_plugin_free(void*, size_t);
extern void cliproxy_plugin_shutdown(void);
*/
import "C"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
	"gopkg.in/yaml.v3"
)

const abiVersion = 1

const (
	pluginName       = "antigravity-coding-filter"
	pluginVersion    = "0.3.4"
	pluginRepository = "https://github.com/jellyfish-p/cpa-plugin-antigravity-coding-filter"
)

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(_ *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return 1
	}
	plugin.abi_version = abiVersion
	plugin.call = (C.cliproxy_plugin_call_fn)(C.cliproxy_plugin_call)
	plugin.free_buffer = (C.cliproxy_plugin_free_fn)(C.cliproxy_plugin_free)
	plugin.shutdown = (C.cliproxy_plugin_shutdown_fn)(C.cliproxy_plugin_shutdown)
	return 0
}

//export cliproxy_plugin_call
func cliproxy_plugin_call(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response != nil {
		response.ptr = nil
		response.len = 0
	}
	if method == nil {
		writeCResponse(response, mustErrorEnvelope("invalid_method", "method is required"))
		return 1
	}

	methodName := C.GoString(method)
	var requestBytes []byte
	if request != nil && requestLen > 0 {
		requestBytes = C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
	}

	raw, code := handlePluginCall(methodName, requestBytes)
	writeCResponse(response, raw)
	return C.int(code)
}

//export cliproxy_plugin_free
func cliproxy_plugin_free(ptr unsafe.Pointer, _ C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
}

//export cliproxy_plugin_shutdown
func cliproxy_plugin_shutdown() {}

func writeCResponse(response *C.cliproxy_buffer, raw []byte) {
	if response == nil || len(raw) == 0 {
		return
	}
	ptr := C.malloc(C.size_t(len(raw)))
	if ptr == nil {
		return
	}
	C.memcpy(ptr, unsafe.Pointer(&raw[0]), C.size_t(len(raw)))
	response.ptr = ptr
	response.len = C.size_t(len(raw))
}

func handlePluginCall(method string, request []byte) ([]byte, int) {
	switch method {
	case pluginabi.MethodPluginRegister:
		return handlePluginLifecycle(request), 0
	case pluginabi.MethodPluginReconfigure:
		return handlePluginLifecycle(request), 0
	case pluginabi.MethodModelRoute:
		return handleModelRoute(request), 0
	case pluginabi.MethodExecutorExecute, pluginabi.MethodExecutorCountTokens, pluginabi.MethodExecutorExecuteStream:
		return blockErrorEnvelope(), 0
	case pluginabi.MethodExecutorHTTPRequest:
		return mustEnvelope(pluginapi.ExecutorHTTPResponse{
			StatusCode: http.StatusForbidden,
			Body:       blockPayload(),
			Headers:    jsonHeaders(),
		}), 0
	case pluginabi.MethodRequestInterceptBefore:
		return handleRequestInterceptBefore(request), 0
	case pluginabi.MethodRequestInterceptAfter:
		return handleRequestInterceptAfter(request), 0
	default:
		return mustErrorEnvelope("unknown_method", fmt.Sprintf("unknown method %q", method)), 0
	}
}

func handlePluginLifecycle(request []byte) []byte {
	if len(request) > 0 {
		cfg, err := filterConfigFromLifecycleRequest(request)
		if err != nil {
			return mustErrorEnvelope("invalid_config", err.Error())
		}
		applyFilterConfig(cfg)
	}
	return mustEnvelope(registrationResponse())
}

func registrationResponse() any {
	return struct {
		SchemaVersion uint32             `json:"schema_version"`
		Metadata      pluginapi.Metadata `json:"metadata"`
		Capabilities  struct {
			ModelRouter           bool                         `json:"model_router"`
			Executor              bool                         `json:"executor"`
			ExecutorModelScope    pluginapi.ExecutorModelScope `json:"executor_model_scope"`
			ExecutorInputFormats  []string                     `json:"executor_input_formats,omitempty"`
			ExecutorOutputFormats []string                     `json:"executor_output_formats,omitempty"`
			RequestInterceptor    bool                         `json:"request_interceptor"`
		} `json:"capabilities"`
	}{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             pluginName,
			Version:          pluginVersion,
			Author:           "local",
			GitHubRepository: pluginRepository,
			Logo:             "",
			ConfigFields:     configFields(),
		},
		Capabilities: struct {
			ModelRouter           bool                         `json:"model_router"`
			Executor              bool                         `json:"executor"`
			ExecutorModelScope    pluginapi.ExecutorModelScope `json:"executor_model_scope"`
			ExecutorInputFormats  []string                     `json:"executor_input_formats,omitempty"`
			ExecutorOutputFormats []string                     `json:"executor_output_formats,omitempty"`
			RequestInterceptor    bool                         `json:"request_interceptor"`
		}{
			ModelRouter:           true,
			Executor:              true,
			ExecutorModelScope:    pluginapi.ExecutorModelScopeBoth,
			ExecutorInputFormats:  []string{"chat-completions", "responses", "anthropic", "gemini"},
			ExecutorOutputFormats: []string{"chat-completions", "responses", "anthropic", "gemini"},
			RequestInterceptor:    true,
		},
	}
}

func configFields() []pluginapi.ConfigField {
	return []pluginapi.ConfigField{
		{
			Name:        "mode",
			Type:        pluginapi.ConfigFieldTypeEnum,
			EnumValues:  []string{string(filterModeBlock), string(filterModeRewrite)},
			Description: "How matched requests are handled: block rejects the request (default); rewrite replaces matched names with Antigravity.",
		},
		{
			Name:        "scope",
			Type:        pluginapi.ConfigFieldTypeArray,
			Description: fmt.Sprintf("Select any combination of %s for block and rewrite modes. Defaults to [\"system\"]. An empty list disables matching.", strings.Join(scopeBlockNames(), ", ")),
		},
		{
			Name:        "use_default_keywords",
			Type:        pluginapi.ConfigFieldTypeBoolean,
			Description: "Enable the built-in coding software and agent keyword preset.",
		},
		{
			Name:        "custom_mappings",
			Type:        pluginapi.ConfigFieldTypeObject,
			Description: "Additional case-insensitive mappings for the selected scope. Block mode checks keys. Rewrite mode replaces keys with their values.",
		},
	}
}

func handleModelRoute(request []byte) []byte {
	// Provider selection must happen before this filter runs.  The after-auth
	// interceptor below is the first hook that knows whether the selected
	// provider is Antigravity, so routing is deliberately left untouched.
	return mustEnvelope(pluginapi.ModelRouteResponse{Handled: false})
}

const (
	blockErrorCode    = "blocked_by_antigravity_coding_filter"
	blockErrorMessage = "request blocked because it matches a configured non-Antigravity coding software keyword"
)

func blockErrorEnvelope() []byte {
	return mustHTTPErrorEnvelope(blockErrorCode, blockErrorMessage, http.StatusForbidden)
}

func blockPayload() []byte {
	payload := map[string]any{
		"error": map[string]any{
			"code":    blockErrorCode,
			"message": blockErrorMessage,
			"type":    "invalid_request_error",
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return []byte(`{"error":{"code":"blocked_by_antigravity_coding_filter"}}`)
	}
	return raw
}

func jsonHeaders() http.Header {
	return http.Header{"content-type": []string{"application/json"}}
}

func handleRequestInterceptBefore(request []byte) []byte {
	var req pluginapi.RequestInterceptRequest
	if err := json.Unmarshal(request, &req); err != nil {
		return mustErrorEnvelope("invalid_request", fmt.Sprintf("decode request.intercept_before request: %v", err))
	}
	return mustEnvelope(pluginapi.RequestInterceptResponse{})
}

func handleRequestInterceptAfter(request []byte) []byte {
	var req pluginapi.RequestInterceptRequest
	if err := json.Unmarshal(request, &req); err != nil {
		return mustErrorEnvelope("invalid_request", fmt.Sprintf("decode request.intercept_after request: %v", err))
	}
	if !strings.EqualFold(strings.TrimSpace(req.ToFormat), "antigravity") {
		return mustEnvelope(pluginapi.RequestInterceptResponse{})
	}

	cfg := activeFilterConfig()
	if cfg.Mode == filterModeBlock {
		decision := classifyRequestWithConfig(req.Body, cfg)
		if !decision.Blocked {
			return mustEnvelope(pluginapi.RequestInterceptResponse{})
		}
		return mustEnvelope(pluginapi.RequestInterceptResponse{
			Terminate:       true,
			StatusCode:      http.StatusForbidden,
			ResponseHeaders: jsonHeaders(),
			ResponseBody:    blockPayload(),
		})
	}

	body, rewritten := rewriteRequestBodyWithConfig(req.Body, cfg)
	if !rewritten {
		return mustEnvelope(pluginapi.RequestInterceptResponse{})
	}
	return mustEnvelope(pluginapi.RequestInterceptResponse{Body: body})
}

func mustEnvelope(result any) []byte {
	raw, err := json.Marshal(pluginabi.Envelope{OK: true, Result: mustRawMessage(result)})
	if err != nil {
		return mustErrorEnvelope("marshal_error", err.Error())
	}
	return raw
}

func mustErrorEnvelope(code, message string) []byte {
	return mustHTTPErrorEnvelope(code, message, 0)
}

func mustHTTPErrorEnvelope(code, message string, status int) []byte {
	raw, err := json.Marshal(pluginabi.Envelope{OK: false, Error: &pluginabi.Error{
		Code:       code,
		Message:    message,
		HTTPStatus: status,
	}})
	if err != nil {
		return []byte(`{"ok":false,"error":{"code":"marshal_error","message":"failed to encode plugin response"}}`)
	}
	return raw
}

func mustRawMessage(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

var defaultRewriteMappings = normalizeMappings([]rewriteMapping{
	// Major AI code editors, assistants, and terminal coding agents.
	{Match: "Claude Agent", Replacement: "Antigravity"},
	{Match: "Claude Code", Replacement: "Antigravity"},
	{Match: "OpenAI Codex", Replacement: "Antigravity"},
	{Match: "Codex CLI", Replacement: "Antigravity"},
	{Match: "Codex", Replacement: "Antigravity"},
	{Match: "OpenCode", Replacement: "Antigravity"},
	{Match: "GitHub Copilot CLI", Replacement: "Antigravity"},
	{Match: "GitHub Copilot", Replacement: "Antigravity"},
	{Match: "Gemini Code Assist", Replacement: "Antigravity"},
	{Match: "Gemini CLI", Replacement: "Antigravity"},
	{Match: "Cursor", Replacement: "Antigravity"},
	{Match: "Windsurf", Replacement: "Antigravity"},
	{Match: "Codeium", Replacement: "Antigravity"},
	{Match: "Cline", Replacement: "Antigravity"},
	{Match: "Roo Code", Replacement: "Antigravity"},
	{Match: "Kilo Code", Replacement: "Antigravity"},
	{Match: "Aider", Replacement: "Antigravity"},
	{Match: "Continue.dev", Replacement: "Antigravity"},
	{Match: "Amazon Q Developer", Replacement: "Antigravity"},
	{Match: "Amazon CodeWhisperer", Replacement: "Antigravity"},
	{Match: "JetBrains AI Assistant", Replacement: "Antigravity"},
	{Match: "JetBrains Junie", Replacement: "Antigravity"},
	{Match: "Kiro", Replacement: "Antigravity"},
	{Match: "Qoder CLI", Replacement: "Antigravity"},
	{Match: "Qoder", Replacement: "Antigravity"},
	{Match: "Qwen Code", Replacement: "Antigravity"},
	{Match: "Trae", Replacement: "Antigravity"},
	{Match: "Tabnine", Replacement: "Antigravity"},
	{Match: "Sourcegraph Cody", Replacement: "Antigravity"},
	{Match: "Augment Code", Replacement: "Antigravity"},
	{Match: "Replit Agent", Replacement: "Antigravity"},
	{Match: "Replit Ghostwriter", Replacement: "Antigravity"},
	{Match: "Devin", Replacement: "Antigravity"},
	{Match: "OpenHands", Replacement: "Antigravity"},
	{Match: "SWE-agent", Replacement: "Antigravity"},
	{Match: "Goose", Replacement: "Antigravity"},
	{Match: "Zed AI", Replacement: "Antigravity"},
	{Match: "Void Editor", Replacement: "Antigravity"},
	{Match: "PearAI", Replacement: "Antigravity"},
	{Match: "Refact.ai", Replacement: "Antigravity"},
	{Match: "Tabby", Replacement: "Antigravity"},
	{Match: "GitLab Duo", Replacement: "Antigravity"},
	{Match: "Visual Studio IntelliCode", Replacement: "Antigravity"},
	{Match: "CodeBuddy", Replacement: "Antigravity"},
	{Match: "Blackbox AI", Replacement: "Antigravity"},
	{Match: "Pieces for Developers", Replacement: "Antigravity"},
	{Match: "Qodo", Replacement: "Antigravity"},
	{Match: "CodiumAI", Replacement: "Antigravity"},
	{Match: "Rovo Dev CLI", Replacement: "Antigravity"},
	{Match: "Factory Droid", Replacement: "Antigravity"},

	// General-purpose local agents that can generate and modify code.
	{Match: "OpenClaw", Replacement: "Antigravity"},
	{Match: "Clawdbot", Replacement: "Antigravity"},
	{Match: "Moltbot", Replacement: "Antigravity"},
	{Match: "Hermes Agent", Replacement: "Antigravity"},
	{Match: "Hermes", Replacement: "Antigravity"},
	{Match: "WorkBuddy", Replacement: "Antigravity"},
})

var defaultSystemPromptMappings = normalizeMappings([]rewriteMapping{
	// DeepSeek Harness's default persona and Pi's current and earlier preambles.
	{Match: "powered by DeepSeek Harness", Replacement: "powered by Antigravity"},
	{Match: "inside pi, a coding agent harness", Replacement: "inside Antigravity, a coding agent harness"},
})

type filterMode string

const (
	filterModeBlock   filterMode = "block"
	filterModeRewrite filterMode = "rewrite"
)

type scopeBlock string

const (
	scopeSystem           scopeBlock = "system"
	scopeUser             scopeBlock = "user"
	scopeAssistant        scopeBlock = "assistant"
	scopeToolResults      scopeBlock = "tool_results"
	scopeToolDescriptions scopeBlock = "tool_descriptions"
)

var scopeBlocks = []scopeBlock{scopeSystem, scopeUser, scopeAssistant, scopeToolResults, scopeToolDescriptions}

func scopeBlockNames() []string {
	names := make([]string, 0, len(scopeBlocks))
	for _, block := range scopeBlocks {
		names = append(names, string(block))
	}
	return names
}

type rewriteMapping struct {
	Match       string
	Replacement string
	pattern     *regexp.Regexp
}

type filterConfig struct {
	Mode               filterMode
	Scope              []scopeBlock
	UseDefaultKeywords bool
	CustomMappings     []rewriteMapping
}

var (
	filterConfigMu      sync.RWMutex
	currentFilterConfig = defaultFilterConfig()
)

func defaultFilterConfig() filterConfig {
	return filterConfig{Mode: filterModeBlock, UseDefaultKeywords: true}
}

func applyFilterConfig(cfg filterConfig) {
	filterConfigMu.Lock()
	defer filterConfigMu.Unlock()

	currentFilterConfig = filterConfig{
		Mode:               cfg.Mode,
		Scope:              slices.Clone(cfg.Scope),
		UseDefaultKeywords: cfg.UseDefaultKeywords,
		CustomMappings:     append([]rewriteMapping(nil), normalizeMappings(cfg.CustomMappings)...),
	}
}

func activeFilterConfig() filterConfig {
	filterConfigMu.RLock()
	defer filterConfigMu.RUnlock()

	return filterConfig{
		Mode:               currentFilterConfig.Mode,
		Scope:              slices.Clone(currentFilterConfig.Scope),
		UseDefaultKeywords: currentFilterConfig.UseDefaultKeywords,
		CustomMappings:     append([]rewriteMapping(nil), currentFilterConfig.CustomMappings...),
	}
}

type lifecycleRequest struct {
	ConfigYAML []byte `json:"config_yaml"`
}

func filterConfigFromLifecycleRequest(request []byte) (filterConfig, error) {
	var req lifecycleRequest
	if err := json.Unmarshal(request, &req); err != nil {
		return filterConfig{}, fmt.Errorf("decode lifecycle request: %w", err)
	}
	return parseFilterConfigYAML(req.ConfigYAML)
}

func parseFilterConfigYAML(raw []byte) (filterConfig, error) {
	cfg := defaultFilterConfig()
	if len(strings.TrimSpace(string(raw))) == 0 {
		return cfg, nil
	}

	var values map[string]any
	if err := yaml.Unmarshal(raw, &values); err != nil {
		return filterConfig{}, fmt.Errorf("decode config yaml: %w", err)
	}
	if value, exists := values["mode"]; exists {
		text, ok := value.(string)
		if !ok {
			return filterConfig{}, fmt.Errorf("mode must be a string")
		}
		cfg.Mode = filterMode(strings.ToLower(strings.TrimSpace(text)))
		if cfg.Mode != filterModeBlock && cfg.Mode != filterModeRewrite {
			return filterConfig{}, fmt.Errorf("mode must be one of: block, rewrite")
		}
	}
	scopeKey := "scope"
	scopeValue, hasScope := values[scopeKey]
	if !hasScope {
		scopeKey = "rewrite_scope"
		scopeValue, hasScope = values[scopeKey]
	}
	if hasScope {
		scope, err := parseScope(scopeKey, scopeValue)
		if err != nil {
			return filterConfig{}, err
		}
		// The legacy setting only selected fields in rewrite mode.
		if scopeKey == "scope" || cfg.Mode == filterModeRewrite {
			cfg.Scope = scope
		}
	}
	if value, exists := values["use_default_keywords"]; exists {
		boolValue, ok := value.(bool)
		if !ok {
			return filterConfig{}, fmt.Errorf("use_default_keywords must be a boolean")
		}
		cfg.UseDefaultKeywords = boolValue
	}
	if value, exists := values["custom_mappings"]; exists {
		mappings, err := parseCustomMappings(value)
		if err != nil {
			return filterConfig{}, err
		}
		cfg.CustomMappings = mappings
	}
	return cfg, nil
}

func parseScope(key string, value any) ([]scopeBlock, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array", key)
	}
	scope := make([]scopeBlock, 0, len(items))
	for _, item := range items {
		name, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%s entries must be strings", key)
		}
		block := scopeBlock(strings.ToLower(strings.TrimSpace(name)))
		if !slices.Contains(scopeBlocks, block) {
			return nil, fmt.Errorf("unknown %s entry %q, choose from %s", key, name, strings.Join(scopeBlockNames(), ", "))
		}
		if !slices.Contains(scope, block) {
			scope = append(scope, block)
		}
	}
	return scope, nil
}

func parseCustomMappings(value any) ([]rewriteMapping, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case string:
		return parseMappingString(typed)
	case map[string]any:
		mappings := make([]rewriteMapping, 0, len(typed))
		for match, replacement := range typed {
			text, ok := replacement.(string)
			if !ok {
				return nil, fmt.Errorf("custom_mappings values must be strings")
			}
			mappings = append(mappings, rewriteMapping{Match: match, Replacement: text})
		}
		return mappings, nil
	case []any:
		mappings := make([]rewriteMapping, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("custom_mappings entries must be strings")
			}
			parsed, err := parseMappingString(text)
			if err != nil {
				return nil, err
			}
			mappings = append(mappings, parsed...)
		}
		return mappings, nil
	default:
		return nil, fmt.Errorf("custom_mappings must be an object, array, or string")
	}
}

func parseMappingString(value string) ([]rewriteMapping, error) {
	entries := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
	mappings := make([]rewriteMapping, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		match, replacement, ok := strings.Cut(entry, ":")
		if !ok {
			return nil, fmt.Errorf("custom_mappings entries must use match: replacement")
		}
		mappings = append(mappings, rewriteMapping{Match: match, Replacement: replacement})
	}
	return mappings, nil
}

func effectiveMappings(cfg filterConfig, block scopeBlock) []rewriteMapping {
	if !cfg.UseDefaultKeywords {
		return normalizeMappings(cfg.CustomMappings)
	}
	if len(cfg.CustomMappings) == 0 && block != scopeSystem {
		return defaultRewriteMappings
	}
	mappings := slices.Clone(defaultRewriteMappings)
	if block == scopeSystem {
		mappings = append(mappings, defaultSystemPromptMappings...)
	}
	mappings = append(mappings, cfg.CustomMappings...)
	return normalizeMappings(mappings)
}

func normalizeMappings(mappings []rewriteMapping) []rewriteMapping {
	seen := make(map[string]struct{}, len(mappings))
	reversed := make([]rewriteMapping, 0, len(mappings))
	for i := len(mappings) - 1; i >= 0; i-- {
		mapping := mappings[i]
		match := strings.TrimSpace(mapping.Match)
		replacement := strings.TrimSpace(mapping.Replacement)
		if match == "" || replacement == "" {
			continue
		}
		key := strings.ToLower(match)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		mapping.Match = key
		mapping.Replacement = replacement
		if mapping.pattern == nil {
			mapping.pattern = literalMatchPattern(match)
		}
		reversed = append(reversed, mapping)
	}
	out := make([]rewriteMapping, 0, len(reversed))
	for i := len(reversed) - 1; i >= 0; i-- {
		out = append(out, reversed[i])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i].Match) == len(out[j].Match) {
			return out[i].Match < out[j].Match
		}
		return len(out[i].Match) > len(out[j].Match)
	})
	return out
}

type filterDecision struct {
	Blocked bool
	Signal  string
	Detail  string
}

func classifyRequest(body []byte) filterDecision {
	return classifyRequestWithConfig(body, activeFilterConfig())
}

func classifyRequestWithConfig(body []byte, cfg filterConfig) filterDecision {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return filterDecision{}
	}

	var decision filterDecision
	transforms := scopeTransforms(cfg, func(block scopeBlock, text string, mappings []rewriteMapping) (string, bool) {
		if decision.Blocked {
			return text, false
		}
		if mapping, found := findEligibleMapping(text, mappings); found {
			decision = filterDecision{Blocked: true, Signal: string(block) + ".keyword", Detail: mapping.Match}
		}
		return text, false
	})
	if len(transforms) > 0 {
		transformPromptObject(root, transforms)
	}
	return decision
}

func rewriteRequestBody(body []byte) ([]byte, bool) {
	return rewriteRequestBodyWithConfig(body, activeFilterConfig())
}

func rewriteRequestBodyWithConfig(body []byte, cfg filterConfig) ([]byte, bool) {
	var root any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return nil, false
	}
	transforms := scopeTransforms(cfg, func(_ scopeBlock, text string, mappings []rewriteMapping) (string, bool) {
		return rewritePromptText(text, mappings)
	})
	if len(transforms) == 0 || !transformPromptObject(root, transforms) {
		return nil, false
	}
	raw, err := json.Marshal(root)
	if err != nil {
		return nil, false
	}
	return raw, true
}

func scopeTransforms(cfg filterConfig, visit func(scopeBlock, string, []rewriteMapping) (string, bool)) map[scopeBlock]promptTransform {
	scope := cfg.Scope
	if scope == nil {
		scope = []scopeBlock{scopeSystem}
	}
	transforms := make(map[scopeBlock]promptTransform, len(scope))
	for _, block := range scope {
		mappings := effectiveMappings(cfg, block)
		if len(mappings) > 0 {
			transforms[block] = func(text string) (string, bool) {
				return visit(block, text, mappings)
			}
		}
	}
	return transforms
}

// These helpers keep the original classifier/rewrite seams available to
// callers that used the pre-protocol implementation directly. The runtime
// entry points above use the narrower protocol-aware traversal.
func rewriteSystemFields(value any, mappings []rewriteMapping) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		changed := false
		for key, child := range typed {
			var next any
			var childChanged bool
			if key == "system" {
				next, childChanged = rewriteSystemValue(child, mappings)
			} else {
				next, childChanged = rewriteSystemFields(child, mappings)
			}
			if childChanged {
				typed[key] = next
				changed = true
			}
		}
		return typed, changed
	case []any:
		changed := false
		for index, child := range typed {
			next, childChanged := rewriteSystemFields(child, mappings)
			if childChanged {
				typed[index] = next
				changed = true
			}
		}
		return typed, changed
	default:
		return value, false
	}
}

func rewriteSystemValue(value any, mappings []rewriteMapping) (any, bool) {
	switch typed := value.(type) {
	case string:
		return rewritePromptText(typed, mappings)
	case map[string]any:
		changed := false
		for key, child := range typed {
			next, childChanged := rewriteSystemValue(child, mappings)
			if childChanged {
				typed[key] = next
				changed = true
			}
		}
		return typed, changed
	case []any:
		changed := false
		for index, child := range typed {
			next, childChanged := rewriteSystemValue(child, mappings)
			if childChanged {
				typed[index] = next
				changed = true
			}
		}
		return typed, changed
	default:
		return value, false
	}
}

type promptTransform func(string) (string, bool)

// The traversal follows protocol-defined text fields and the Antigravity
// request wrapper, preserving structural fields such as tool-call identifiers.
func transformPromptObject(value any, transforms map[scopeBlock]promptTransform) bool {
	object, ok := value.(map[string]any)
	if !ok {
		return false
	}
	changed := false
	if transform := transforms[scopeSystem]; transform != nil {
		for _, key := range []string{"system", "instructions"} {
			child, exists := object[key]
			if !exists {
				continue
			}
			next, childChanged := transformPromptTextContainer(child, transform)
			if childChanged {
				object[key] = next
				changed = true
			}
		}
		for _, key := range []string{"systemInstruction", "system_instruction"} {
			child, exists := object[key]
			if !exists {
				continue
			}
			next, childChanged := transformGeminiSystemInstruction(child, transform)
			if childChanged {
				object[key] = next
				changed = true
			}
		}
	}

	if transformTextField(object, "input", transforms[scopeUser]) {
		changed = true
	}
	for _, field := range []string{"input", "messages"} {
		if transformRoleMessages(object[field], transforms) {
			changed = true
		}
	}
	if transformGeminiMessages(object["contents"], transforms) {
		changed = true
	}
	if transformToolDescriptions(object, transforms[scopeToolDescriptions]) {
		changed = true
	}
	if transformPromptObject(object["request"], transforms) {
		changed = true
	}
	return changed
}

func transformPromptTextContainer(value any, transform func(string) (string, bool)) (any, bool) {
	switch typed := value.(type) {
	case string:
		return transform(typed)
	case []any:
		changed := false
		for index, child := range typed {
			next, childChanged := transformPromptTextContainer(child, transform)
			if childChanged {
				typed[index] = next
				changed = true
			}
		}
		return typed, changed
	case map[string]any:
		// Anthropic and Responses text blocks carry their prompt in `text`.
		if text, ok := typed["text"].(string); ok {
			next, childChanged := transform(text)
			if childChanged {
				typed["text"] = next
			}
			return typed, childChanged
		}
	}
	return value, false
}

func transformRoleMessages(value any, transforms map[scopeBlock]promptTransform) bool {
	switch typed := value.(type) {
	case []any:
		changed := false
		for _, child := range typed {
			if transformRoleMessage(child, transforms) {
				changed = true
			}
		}
		return changed
	case map[string]any:
		return transformRoleMessage(typed, transforms)
	default:
		return false
	}
}

func transformRoleMessage(value any, transforms map[scopeBlock]promptTransform) bool {
	message, ok := value.(map[string]any)
	if !ok {
		return false
	}
	if itemType := normalizedJSONField(message, "type"); itemType == "function_call_output" || itemType == "custom_tool_call_output" {
		return transformContentField(message, "output", transforms[scopeToolResults], false)
	}
	switch normalizedJSONField(message, "role") {
	case "system", "developer":
		return transformContentField(message, "content", transforms[scopeSystem], false)
	case "assistant":
		return transformContentField(message, "content", transforms[scopeAssistant], true)
	case "tool", "function":
		return transformContentField(message, "content", transforms[scopeToolResults], false)
	case "user":
		changed := transformContentField(message, "content", transforms[scopeUser], false)
		if transform := transforms[scopeToolResults]; transform != nil {
			blocks, _ := message["content"].([]any)
			for _, child := range blocks {
				block, _ := child.(map[string]any)
				if normalizedJSONField(block, "type") == "tool_result" && transformContentField(block, "content", transform, false) {
					changed = true
				}
			}
		}
		return changed
	default:
		return false
	}
}

func transformContentField(object map[string]any, field string, transform promptTransform, allowOutputText bool) bool {
	if transform == nil {
		return false
	}
	next, changed := transformMessageContent(object[field], transform, allowOutputText)
	if changed {
		object[field] = next
	}
	return changed
}

func transformMessageContent(value any, transform promptTransform, allowOutputText bool) (any, bool) {
	switch typed := value.(type) {
	case string:
		return transform(typed)
	case []any:
		changed := false
		for _, child := range typed {
			block, _ := child.(map[string]any)
			if transformMessageTextBlock(block, transform, allowOutputText) {
				changed = true
			}
		}
		return typed, changed
	case map[string]any:
		return typed, transformMessageTextBlock(typed, transform, allowOutputText)
	default:
		return value, false
	}
}

func normalizedJSONField(object map[string]any, field string) string {
	value, _ := object[field].(string)
	return strings.ToLower(strings.TrimSpace(value))
}

func transformTextField(object map[string]any, field string, transform promptTransform) bool {
	text, ok := object[field].(string)
	if !ok || transform == nil {
		return false
	}
	next, changed := transform(text)
	if changed {
		object[field] = next
	}
	return changed
}

func transformToolDescriptions(request map[string]any, transform promptTransform) bool {
	if transform == nil {
		return false
	}

	changed := transformFunctionDescriptions(request["functions"], transform)
	tools, _ := request["tools"].([]any)
	for _, entry := range tools {
		tool, _ := entry.(map[string]any)
		if transformTextField(tool, "description", transform) {
			changed = true
		}
		function, _ := tool["function"].(map[string]any)
		if transformTextField(function, "description", transform) {
			changed = true
		}
		for _, key := range []string{"functionDeclarations", "function_declarations"} {
			if transformFunctionDescriptions(tool[key], transform) {
				changed = true
			}
		}
	}
	return changed
}

func transformFunctionDescriptions(value any, transform promptTransform) bool {
	functions, _ := value.([]any)
	changed := false
	for _, entry := range functions {
		function, _ := entry.(map[string]any)
		if transformTextField(function, "description", transform) {
			changed = true
		}
	}
	return changed
}

func transformMessageTextBlock(block map[string]any, transform promptTransform, allowOutputText bool) bool {
	switch normalizedJSONField(block, "type") {
	case "text", "input_text":
		return transformTextField(block, "text", transform)
	case "output_text":
		return allowOutputText && transformTextField(block, "text", transform)
	default:
		return false
	}
}

func transformGeminiMessages(value any, transforms map[scopeBlock]promptTransform) bool {
	userTransform := transforms[scopeUser]
	assistantTransform := transforms[scopeAssistant]
	toolTransform := transforms[scopeToolResults]
	if userTransform == nil && assistantTransform == nil && toolTransform == nil {
		return false
	}
	messages, _ := value.([]any)
	changed := false
	for _, child := range messages {
		message, _ := child.(map[string]any)
		parts, _ := message["parts"].([]any)
		var transform promptTransform
		switch normalizedJSONField(message, "role") {
		case "", "user":
			transform = userTransform
		case "assistant", "model":
			transform = assistantTransform
		case "tool", "function":
			transform = toolTransform
		}
		for _, child := range parts {
			part, _ := child.(map[string]any)
			if thought, _ := part["thought"].(bool); !thought && transformTextField(part, "text", transform) {
				changed = true
			}
			if toolTransform != nil {
				for _, field := range []string{"functionResponse", "function_response"} {
					response, _ := part[field].(map[string]any)
					if transformGeminiFunctionResponse(response, toolTransform) {
						changed = true
					}
				}
			}
		}
	}
	return changed
}

func transformGeminiFunctionResponse(response map[string]any, transform promptTransform) bool {
	if transformTextField(response, "response", transform) {
		return true
	}
	result, _ := response["response"].(map[string]any)
	changed := false
	for _, field := range []string{"result", "output", "error", "text"} {
		if transformContentField(result, field, transform, false) {
			changed = true
		}
	}
	parts, _ := response["parts"].([]any)
	for _, child := range parts {
		part, _ := child.(map[string]any)
		if transformTextField(part, "text", transform) {
			changed = true
		}
	}
	return changed
}

func transformGeminiSystemInstruction(value any, transform func(string) (string, bool)) (any, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		return value, false
	}
	parts, ok := object["parts"].([]any)
	if !ok {
		return value, false
	}
	changed := false
	for index, part := range parts {
		partObject, ok := part.(map[string]any)
		if !ok {
			continue
		}
		text, ok := partObject["text"].(string)
		if !ok {
			continue
		}
		next, partChanged := transform(text)
		if partChanged {
			partObject["text"] = next
			parts[index] = partObject
			changed = true
		}
	}
	return object, changed
}

func findEligibleMapping(text string, mappings []rewriteMapping) (rewriteMapping, bool) {
	for _, mapping := range mappings {
		if len(unprotectedMatches(text, mapping.matchPattern())) > 0 {
			return mapping, true
		}
	}
	return rewriteMapping{}, false
}

func rewritePromptText(value string, mappings []rewriteMapping) (string, bool) {
	next := value
	changed := false
	for _, mapping := range mappings {
		var replaced bool
		next, replaced = replaceMatchedText(next, mapping.matchPattern(), mapping.Replacement)
		changed = changed || replaced
	}
	return next, changed
}

func (mapping rewriteMapping) matchPattern() *regexp.Regexp {
	if mapping.pattern != nil {
		return mapping.pattern
	}
	return literalMatchPattern(mapping.Match)
}

func containsUnprotected(value, match string) bool {
	return len(unprotectedMatches(value, literalMatchPattern(match))) > 0
}

func replaceInsensitiveProtected(value, match, replacement string) (string, bool) {
	return replaceMatchedText(value, literalMatchPattern(match), replacement)
}

func literalMatchPattern(match string) *regexp.Regexp {
	if match == "" {
		return nil
	}
	pattern, _ := regexp.Compile("(?i)" + regexp.QuoteMeta(match))
	return pattern
}

func unprotectedMatches(value string, pattern *regexp.Regexp) [][]int {
	if pattern == nil {
		return nil
	}
	indices := pattern.FindAllStringIndex(value, -1)
	if len(indices) == 0 {
		return nil
	}
	protected := protectedTextRanges(value)
	return slices.DeleteFunc(indices, func(index []int) bool {
		return overlapsProtected(index[0], index[1], protected)
	})
}

func replaceMatchedText(value string, pattern *regexp.Regexp, replacement string) (string, bool) {
	matches := unprotectedMatches(value, pattern)
	if len(matches) == 0 {
		return value, false
	}
	var builder strings.Builder
	start := 0
	for _, match := range matches {
		builder.WriteString(value[start:match[0]])
		builder.WriteString(replacement)
		start = match[1]
	}
	builder.WriteString(value[start:])
	return builder.String(), true
}

func replaceInsensitive(value, match, replacement string) (string, bool) {
	return replaceInsensitiveProtected(value, match, replacement)
}

type textRange struct {
	start int
	end   int
}

var protectedTextPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:https?|ftp)://[^\s"'<>]+`),
	regexp.MustCompile(`(?i)\bmcp(?:__|\\_\\_)[A-Za-z0-9][A-Za-z0-9_\\-]*`),
	regexp.MustCompile(`(?i)\b[A-Za-z]:[\\/][^\s"'<>]+`),
	regexp.MustCompile(`(?:\\\\|//)[^\s"'<>]+`),
	regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])/(?:[^\s"'<>/]+/)+[^\s"'<>]*`),
	regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])(?:[A-Za-z0-9._~-]+[\\/])+[A-Za-z0-9._~-]+`),
	regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])(?:~|\.{1,2})[\\/][^\s"'<>]+`),
	regexp.MustCompile(`(?i)\.codex(?:[^A-Za-z0-9]|$)`),
	regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])[A-Za-z0-9_-]+\.(?:md|markdown)`),
}

func protectedTextRanges(value string) []textRange {
	var ranges []textRange
	for _, pattern := range protectedTextPatterns {
		for _, match := range pattern.FindAllStringIndex(value, -1) {
			ranges = append(ranges, textRange{start: match[0], end: match[1]})
		}
	}
	return ranges
}

func overlapsProtected(start, end int, ranges []textRange) bool {
	for _, protected := range ranges {
		if start < protected.end && protected.start < end {
			return true
		}
	}
	return false
}

func walkJSON(value any, visit func(path []string, value any) bool) {
	var walk func(path []string, current any) bool
	walk = func(path []string, current any) bool {
		if !visit(path, current) {
			return false
		}
		switch typed := current.(type) {
		case map[string]any:
			for key, child := range typed {
				if !walk(appendPath(path, key), child) {
					return false
				}
			}
		case []any:
			for index, child := range typed {
				if !walk(appendPath(path, fmt.Sprintf("%d", index)), child) {
					return false
				}
			}
		}
		return true
	}
	walk(nil, value)
}

func appendPath(path []string, item string) []string {
	next := make([]string, len(path), len(path)+1)
	copy(next, path)
	return append(next, item)
}

func collectText(value any) string {
	var parts []string
	var collect func(any)
	collect = func(current any) {
		switch typed := current.(type) {
		case string:
			parts = append(parts, typed)
		case map[string]any:
			for _, child := range typed {
				collect(child)
			}
		case []any:
			for _, child := range typed {
				collect(child)
			}
		}
	}
	collect(value)
	return strings.Join(parts, "\n")
}
