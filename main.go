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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unsafe"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const abiVersion = 1

const (
	pluginName    = "antigravity-coding-filter"
	pluginVersion = "0.1.0"
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

type executorStreamResponse struct {
	Headers http.Header           `json:"Headers,omitempty"`
	Chunks  []executorStreamChunk `json:"Chunks,omitempty"`
}

type executorStreamChunk struct {
	Payload []byte `json:"Payload"`
}

func handlePluginCall(method string, request []byte) ([]byte, int) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		return mustEnvelope(registrationResponse()), 0
	case pluginabi.MethodModelRoute:
		return handleModelRoute(request), 0
	case pluginabi.MethodExecutorExecute, pluginabi.MethodExecutorCountTokens:
		return mustEnvelope(pluginapi.ExecutorResponse{
			Payload: blockPayload(),
			Headers: jsonHeaders(),
		}), 0
	case pluginabi.MethodExecutorHTTPRequest:
		return mustEnvelope(pluginapi.ExecutorHTTPResponse{
			StatusCode: http.StatusForbidden,
			Body:       blockPayload(),
			Headers:    jsonHeaders(),
		}), 0
	case pluginabi.MethodExecutorExecuteStream:
		return mustEnvelope(executorStreamResponse{
			Headers: jsonHeaders(),
			Chunks:  []executorStreamChunk{{Payload: blockPayload()}},
		}), 0
	default:
		return mustErrorEnvelope("unknown_method", fmt.Sprintf("unknown method %q", method)), 0
	}
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
		} `json:"capabilities"`
	}{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             pluginName,
			Version:          pluginVersion,
			Author:           "local",
			GitHubRepository: "",
			Logo:             "",
			ConfigFields:     []pluginapi.ConfigField{},
		},
		Capabilities: struct {
			ModelRouter           bool                         `json:"model_router"`
			Executor              bool                         `json:"executor"`
			ExecutorModelScope    pluginapi.ExecutorModelScope `json:"executor_model_scope"`
			ExecutorInputFormats  []string                     `json:"executor_input_formats,omitempty"`
			ExecutorOutputFormats []string                     `json:"executor_output_formats,omitempty"`
		}{
			ModelRouter:           true,
			Executor:              true,
			ExecutorModelScope:    pluginapi.ExecutorModelScopeBoth,
			ExecutorInputFormats:  []string{"chat-completions", "responses", "anthropic", "gemini"},
			ExecutorOutputFormats: []string{"chat-completions", "responses", "anthropic", "gemini"},
		},
	}
}

func handleModelRoute(request []byte) []byte {
	var req pluginapi.ModelRouteRequest
	if err := json.Unmarshal(request, &req); err != nil {
		return mustErrorEnvelope("invalid_request", fmt.Sprintf("decode model.route request: %v", err))
	}

	decision := classifyRequest(req.Body)
	if !decision.Blocked {
		return mustEnvelope(pluginapi.ModelRouteResponse{Handled: false})
	}

	return mustEnvelope(pluginapi.ModelRouteResponse{
		Handled:    true,
		TargetKind: pluginapi.ModelRouteTargetSelf,
		Reason:     fmt.Sprintf("%s:%s", decision.Signal, decision.Detail),
	})
}

func blockPayload() []byte {
	payload := map[string]any{
		"error": map[string]any{
			"code":    "blocked_by_antigravity_coding_filter",
			"message": "request blocked because it matches non-Antigravity coding software signals",
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

func mustEnvelope(result any) []byte {
	raw, err := json.Marshal(pluginabi.Envelope{OK: true, Result: mustRawMessage(result)})
	if err != nil {
		return mustErrorEnvelope("marshal_error", err.Error())
	}
	return raw
}

func mustErrorEnvelope(code, message string) []byte {
	raw, err := json.Marshal(pluginabi.Envelope{OK: false, Error: &pluginabi.Error{Code: code, Message: message}})
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

var defaultCodingKeywords = []string{
	"opencode",
	"codex",
	"claude code",
}

type filterDecision struct {
	Blocked bool
	Signal  string
	Detail  string
}

func classifyRequest(body []byte) filterDecision {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return filterDecision{}
	}
	return classifyJSON(root)
}

func classifyJSON(root any) filterDecision {
	var decision filterDecision
	walkJSON(root, func(path []string, value any) bool {
		key := ""
		if len(path) > 0 {
			key = path[len(path)-1]
		}

		if key == "prompt_cache_key" {
			decision = filterDecision{Blocked: true, Signal: "prompt_cache_key", Detail: strings.Join(path, ".")}
			return false
		}

		if key == "metadata" {
			if object, ok := value.(map[string]any); ok {
				if _, exists := object["user_id"]; exists {
					decision = filterDecision{Blocked: true, Signal: "metadata.user_id", Detail: strings.Join(append(path, "user_id"), ".")}
					return false
				}
			}
		}

		if key == "system" {
			text := strings.ToLower(collectText(value))
			for _, keyword := range defaultCodingKeywords {
				if strings.Contains(text, keyword) {
					decision = filterDecision{Blocked: true, Signal: "system.keyword", Detail: keyword}
					return false
				}
			}
		}

		return true
	})
	return decision
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
