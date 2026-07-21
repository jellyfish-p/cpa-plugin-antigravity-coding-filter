package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestHandlePluginCallRegisterDeclaresBothFilterModes(t *testing.T) {
	raw, code := handlePluginCall("plugin.register", nil)
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope map[string]any
	mustUnmarshalJSON(t, raw, &envelope)
	if envelope["ok"] != true {
		t.Fatalf("ok = %#v, want true", envelope["ok"])
	}
	result := envelope["result"].(map[string]any)
	metadata := result["metadata"].(map[string]any)
	if metadata["GitHubRepository"] != pluginRepository {
		t.Fatalf("GitHubRepository = %#v, want %q", metadata["GitHubRepository"], pluginRepository)
	}
	capabilities := result["capabilities"].(map[string]any)
	if capabilities["model_router"] != true {
		t.Fatalf("model_router = %#v, want true", capabilities["model_router"])
	}
	if capabilities["executor"] != true {
		t.Fatalf("executor = %#v, want true", capabilities["executor"])
	}
	if capabilities["request_interceptor"] != true {
		t.Fatalf("request_interceptor = %#v, want true", capabilities["request_interceptor"])
	}
	fields := result["metadata"].(map[string]any)["ConfigFields"].([]any)
	if !hasEnumConfigField(fields, "mode", []string{"block", "rewrite"}) {
		t.Fatalf("ConfigFields = %#v, want enum mode with block and rewrite", fields)
	}
	if !hasConfigField(fields, "use_default_keywords", "boolean") {
		t.Fatalf("ConfigFields = %#v, want boolean use_default_keywords", fields)
	}
	if !hasConfigField(fields, "custom_mappings", "object") {
		t.Fatalf("ConfigFields = %#v, want object custom_mappings", fields)
	}
}

func TestHandlePluginCallReconfigureAppliesCustomMappingsAndDefaultToggle(t *testing.T) {
	defer restoreDefaultFilterConfig(t)

	raw, code := handlePluginCall("plugin.reconfigure", lifecycleRequestJSON(t, []byte(`
enabled: true
priority: 1
use_default_keywords: false
mode: rewrite
custom_mappings:
  Cursor: Antigravity
  Windsurf: Antigravity
`)))
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	if got, rewritten := rewriteRequestBody([]byte(`{"system":"You are Codex."}`)); rewritten {
		t.Fatalf("Codex rewritten after disabling defaults; body=%s", got)
	}
	got, rewritten := rewriteRequestBody([]byte(`{"system":"route this Cursor session"}`))
	if !rewritten {
		t.Fatalf("Cursor rewritten = false, want true")
	}
	if !strings.Contains(string(got), "route this Antigravity session") {
		t.Fatalf("body = %s, want Cursor replaced with Antigravity", got)
	}
}

func TestHandlePluginCallReconfigureAcceptsDelimitedCustomMappings(t *testing.T) {
	defer restoreDefaultFilterConfig(t)

	raw, code := handlePluginCall("plugin.reconfigure", lifecycleRequestJSON(t, []byte(`
custom_mappings: |
  Cursor: Antigravity
  Windsurf: Antigravity
  JetBrains AI: Antigravity
`)))
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	tests := []string{"Cursor", "Windsurf", "JetBrains AI"}
	for _, keyword := range tests {
		t.Run(keyword, func(t *testing.T) {
			got, rewritten := rewriteRequestBody([]byte(`{"system":"route this ` + keyword + ` session"}`))
			if !rewritten {
				t.Fatalf("%s rewritten = false, want true", keyword)
			}
			if !strings.Contains(string(got), "Antigravity") {
				t.Fatalf("body = %s, want replacement", got)
			}
		})
	}
}

func TestHandlePluginCallReconfigureKeepsPreviousConfigOnInvalidInput(t *testing.T) {
	defer restoreDefaultFilterConfig(t)

	raw, code := handlePluginCall("plugin.reconfigure", lifecycleRequestJSON(t, []byte(`
use_default_keywords: false
custom_mappings:
  Cursor: Antigravity
`)))
	if code != 0 {
		t.Fatalf("initial reconfigure code = %d, want 0; body=%s", code, raw)
	}

	raw, code = handlePluginCall("plugin.reconfigure", lifecycleRequestJSON(t, []byte(`
custom_mappings:
  - nested: invalid
`)))
	if code != 0 {
		t.Fatalf("invalid reconfigure code = %d, want 0 handled error envelope; body=%s", code, raw)
	}

	var envelope struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if envelope.OK {
		t.Fatalf("ok = true, want false")
	}
	if envelope.Error.Code != "invalid_config" {
		t.Fatalf("error code = %q, want invalid_config", envelope.Error.Code)
	}

	if _, rewritten := rewriteRequestBody([]byte(`{"system":"route this Cursor session"}`)); !rewritten {
		t.Fatalf("Cursor rewritten = false after invalid config, want previous config retained")
	}
	if got, rewritten := rewriteRequestBody([]byte(`{"system":"You are Codex."}`)); rewritten {
		t.Fatalf("Codex rewritten after invalid config, want previous disabled-default state retained; body=%s", got)
	}
}

func TestHandlePluginCallRequestInterceptBeforeRewritesCodingSignals(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(filterConfig{Mode: filterModeRewrite, UseDefaultKeywords: true})

	request := requestInterceptRequestJSON(t, `{"system":"You are Codex.","messages":[]}`)

	raw, code := handlePluginCall("request.intercept_before", request)
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Body string `json:"Body"`
		} `json:"result"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if !envelope.OK {
		t.Fatalf("ok = false, want true")
	}
	body, err := base64.StdEncoding.DecodeString(envelope.Result.Body)
	if err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !strings.Contains(string(body), "You are Antigravity.") {
		t.Fatalf("body = %s, want rewritten system", body)
	}
}

func TestHandlePluginCallRequestInterceptBeforeDoesNotRewriteInDefaultBlockMode(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(defaultFilterConfig())

	request := requestInterceptRequestJSON(t, `{"system":"You are Codex.","messages":[]}`)
	raw, code := handlePluginCall("request.intercept_before", request)
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Body string `json:"Body"`
		} `json:"result"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if !envelope.OK || envelope.Result.Body != "" {
		t.Fatalf("response = %s, want unchanged request in block mode", raw)
	}
}

func TestHandlePluginCallModelRouteBlocksMatchedRequestsByDefault(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(defaultFilterConfig())

	request := modelRouteRequestJSON(t, `{"system":"You are Qoder.","messages":[]}`)
	raw, code := handlePluginCall("model.route", request)
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Handled    bool   `json:"Handled"`
			TargetKind string `json:"TargetKind"`
			Reason     string `json:"Reason"`
		} `json:"result"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if !envelope.OK || !envelope.Result.Handled {
		t.Fatalf("response = %s, want handled block route", raw)
	}
	if envelope.Result.TargetKind != "self" {
		t.Fatalf("TargetKind = %q, want self", envelope.Result.TargetKind)
	}
	if !strings.Contains(envelope.Result.Reason, "system.keyword:qoder") {
		t.Fatalf("Reason = %q, want qoder keyword detail", envelope.Result.Reason)
	}
}

func TestHandlePluginCallModelRoutePassesInRewriteMode(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(filterConfig{Mode: filterModeRewrite, UseDefaultKeywords: true})

	raw, code := handlePluginCall("model.route", modelRouteRequestJSON(t, `{"system":"You are Codex."}`))
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Handled bool `json:"Handled"`
		} `json:"result"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if !envelope.OK || envelope.Result.Handled {
		t.Fatalf("response = %s, want unhandled route in rewrite mode", raw)
	}
}

func TestHandlePluginCallExecutorMethodsReturnForbiddenError(t *testing.T) {
	methods := []string{"executor.execute", "executor.execute_stream", "executor.count_tokens"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			raw, code := handlePluginCall(method, []byte(`{"Model":"antigravity/test"}`))
			if code != 0 {
				t.Fatalf("code = %d, want 0; body=%s", code, raw)
			}

			var envelope struct {
				OK    bool `json:"ok"`
				Error struct {
					Code       string `json:"code"`
					Message    string `json:"message"`
					HTTPStatus int    `json:"http_status"`
				} `json:"error"`
			}
			mustUnmarshalJSON(t, raw, &envelope)
			if envelope.OK {
				t.Fatalf("response = %s, want error envelope", raw)
			}
			if envelope.Error.Code != blockErrorCode || envelope.Error.Message != blockErrorMessage {
				t.Fatalf("error = %#v, want configured block error", envelope.Error)
			}
			if envelope.Error.HTTPStatus != 403 {
				t.Fatalf("http_status = %d, want 403", envelope.Error.HTTPStatus)
			}
		})
	}
}

func TestHandlePluginCallExecutorHTTPRequestReturnsForbiddenResponse(t *testing.T) {
	raw, code := handlePluginCall("executor.http_request", []byte(`{}`))
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			StatusCode int                 `json:"StatusCode"`
			Body       string              `json:"Body"`
			Headers    map[string][]string `json:"Headers"`
		} `json:"result"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	body, err := base64.StdEncoding.DecodeString(envelope.Result.Body)
	if err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !envelope.OK || envelope.Result.StatusCode != 403 {
		t.Fatalf("response = %s, want HTTP 403", raw)
	}
	if !strings.Contains(string(body), blockErrorCode) {
		t.Fatalf("body = %s, want block error code", body)
	}
	if got := envelope.Result.Headers["content-type"]; len(got) != 1 || got[0] != "application/json" {
		t.Fatalf("content-type = %#v, want application/json", got)
	}
}

func TestHandlePluginCallReconfigureRejectsInvalidModeAndKeepsPreviousConfig(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(filterConfig{Mode: filterModeRewrite, UseDefaultKeywords: true})

	raw, code := handlePluginCall("plugin.reconfigure", lifecycleRequestJSON(t, []byte("mode: delete\n")))
	if code != 0 {
		t.Fatalf("code = %d, want handled error envelope; body=%s", code, raw)
	}
	var envelope struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if envelope.OK || envelope.Error.Code != "invalid_config" {
		t.Fatalf("response = %s, want invalid_config", raw)
	}
	if got := activeFilterConfig().Mode; got != filterModeRewrite {
		t.Fatalf("mode = %q, want previous rewrite mode retained", got)
	}
}

func TestHandlePluginCallRequestInterceptBeforePassesCleanRequests(t *testing.T) {
	request := requestInterceptRequestJSON(t, `{"system":"You are Antigravity.","messages":[]}`)

	raw, code := handlePluginCall("request.intercept_before", request)
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Body string `json:"Body"`
		} `json:"result"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if !envelope.OK {
		t.Fatalf("ok = false, want true")
	}
	if envelope.Result.Body != "" {
		t.Fatalf("Body = %q, want empty body to keep original request", envelope.Result.Body)
	}
}

func TestHandlePluginCallUnknownMethodReturnsErrorEnvelope(t *testing.T) {
	raw, code := handlePluginCall("unknown.method", nil)
	if code != 0 {
		t.Fatalf("code = %d, want 0 for handled error envelope; body=%s", code, raw)
	}

	var envelope struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if envelope.OK {
		t.Fatalf("ok = true, want false")
	}
	if envelope.Error.Code != "unknown_method" {
		t.Fatalf("error code = %q, want unknown_method", envelope.Error.Code)
	}
}

func requestInterceptRequestJSON(t *testing.T, body string) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"SourceFormat":   "openai",
		"ToFormat":       "",
		"Model":          "antigravity/test",
		"RequestedModel": "antigravity/test",
		"Body":           []byte(body),
	})
	if err != nil {
		t.Fatalf("marshal request intercept request: %v", err)
	}
	return raw
}

func modelRouteRequestJSON(t *testing.T, body string) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"SourceFormat":   "openai",
		"RequestedModel": "antigravity/test",
		"Body":           []byte(body),
	})
	if err != nil {
		t.Fatalf("marshal model route request: %v", err)
	}
	return raw
}

func lifecycleRequestJSON(t *testing.T, configYAML []byte) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"schema_version": 1,
		"config_yaml":    configYAML,
	})
	if err != nil {
		t.Fatalf("marshal lifecycle request: %v", err)
	}
	return raw
}

func hasConfigField(fields []any, name, fieldType string) bool {
	for _, field := range fields {
		object, ok := field.(map[string]any)
		if !ok {
			continue
		}
		if object["Name"] == name && object["Type"] == fieldType {
			return true
		}
	}
	return false
}

func hasEnumConfigField(fields []any, name string, wantValues []string) bool {
	for _, field := range fields {
		object, ok := field.(map[string]any)
		if !ok || object["Name"] != name || object["Type"] != "enum" {
			continue
		}
		values, ok := object["EnumValues"].([]any)
		if !ok || len(values) != len(wantValues) {
			return false
		}
		for i, want := range wantValues {
			if values[i] != want {
				return false
			}
		}
		return true
	}
	return false
}

func restoreDefaultFilterConfig(t *testing.T) {
	t.Helper()
	applyFilterConfig(defaultFilterConfig())
}

func mustUnmarshalJSON(t *testing.T, raw []byte, out any) {
	t.Helper()
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
}
