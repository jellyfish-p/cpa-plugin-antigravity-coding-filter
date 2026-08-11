package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

// --- body coverage: instructions / systemInstruction / role messages ---

func TestRewriteInstructionsField(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(filterConfig{Mode: filterModeRewrite, UseDefaultKeywords: true})

	got, rewritten := rewriteRequestBody([]byte(`{"instructions":"You are Codex, a coding agent.","input":"fix the bug"}`))
	if !rewritten {
		t.Fatal("rewritten = false, want true")
	}
	if !strings.Contains(string(got), "You are Antigravity, a coding agent.") {
		t.Fatalf("body = %s, want instructions rewritten", got)
	}
	if !strings.Contains(string(got), `"input":"fix the bug"`) {
		t.Fatalf("body = %s, want plain string input untouched", got)
	}
}

func TestRewriteGeminiSystemInstruction(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(filterConfig{Mode: filterModeRewrite, UseDefaultKeywords: true})

	body := `{"systemInstruction":{"role":"user","parts":[{"text":"You are Gemini CLI."}]},"contents":[{"role":"user","parts":[{"text":"Gemini CLI?"}]}]}`
	got, rewritten := rewriteRequestBody([]byte(body))
	if !rewritten {
		t.Fatal("rewritten = false, want true")
	}
	if !strings.Contains(string(got), "You are Antigravity.") {
		t.Fatalf("body = %s, want systemInstruction rewritten", got)
	}
	if !strings.Contains(string(got), `"text":"Gemini CLI?"`) {
		t.Fatalf("body = %s, want user contents untouched", got)
	}
}

func TestRewriteMessagesRoleSystemAndDeveloper(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(filterConfig{
		Mode:               filterModeRewrite,
		UseDefaultKeywords: true,
		CustomMappings:     []rewriteMapping{{Match: "Kimi Code", Replacement: "Antigravity"}},
	})

	body := `{"messages":[` +
		`{"role":"system","content":"You are Kimi Code."},` +
		`{"role":"developer","content":[{"type":"text","text":"Act as Codex."}]},` +
		`{"role":"user","content":"compare Codex and Kimi Code"}` +
		`]}`
	got, rewritten := rewriteRequestBody([]byte(body))
	if !rewritten {
		t.Fatal("rewritten = false, want true")
	}
	if !strings.Contains(string(got), "You are Antigravity.") {
		t.Fatalf("body = %s, want system message rewritten", got)
	}
	if !strings.Contains(string(got), "Act as Antigravity.") {
		t.Fatalf("body = %s, want developer message rewritten", got)
	}
	if !strings.Contains(string(got), "compare Codex and Kimi Code") {
		t.Fatalf("body = %s, want user message untouched", got)
	}
}

func TestRewriteResponsesInputRoleDeveloper(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(filterConfig{Mode: filterModeRewrite, UseDefaultKeywords: true})

	body := `{"input":[` +
		`{"type":"message","role":"developer","content":[{"type":"input_text","text":"You are Codex."}]},` +
		`{"type":"message","role":"user","content":[{"type":"input_text","text":"Codex?"}]}` +
		`]}`
	got, rewritten := rewriteRequestBody([]byte(body))
	if !rewritten {
		t.Fatal("rewritten = false, want true")
	}
	if !strings.Contains(string(got), "You are Antigravity.") {
		t.Fatalf("body = %s, want developer input rewritten", got)
	}
	if !strings.Contains(string(got), `"text":"Codex?"`) {
		t.Fatalf("body = %s, want user input untouched", got)
	}
}

func TestRewriteDoesNotTouchToolsOrModel(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(filterConfig{Mode: filterModeRewrite, UseDefaultKeywords: true})

	body := `{"model":"codex-mini","tools":[{"name":"shell","description":"Codex shell tool"}],"messages":[{"role":"user","content":"hi"}]}`
	got, rewritten := rewriteRequestBody([]byte(body))
	if rewritten {
		t.Fatalf("rewritten = true, want false; body=%s", got)
	}
}

// --- header coverage ---

func TestInterceptBeforeRewritesWhitelistedHeadersOnly(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(filterConfig{
		Mode:               filterModeRewrite,
		UseDefaultKeywords: true,
		RewriteHeaders:     true,
		HeaderMappings:     []rewriteMapping{{Match: "kimi-code-cli", Replacement: "Antigravity"}},
	})

	request := interceptRequestWithHeaders(t, `{"system":"plain"}`, map[string][]string{
		"User-Agent":       {"kimi-code-cli/0.34.0"},
		"X-Msh-Platform":   {"kimi-code-cli"},
		"Authorization":    {"Bearer kimi-code-cli-token"},
		"Anthropic-Version": {"2023-06-01"},
	})

	raw, code := handlePluginCall("request.intercept_before", request)
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Body    string              `json:"Body"`
			Headers map[string][]string `json:"Headers"`
		} `json:"result"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if !envelope.OK {
		t.Fatalf("ok = false, want true")
	}
	if envelope.Result.Body != "" {
		t.Fatalf("Body = %q, want empty (no body keywords)", envelope.Result.Body)
	}
	if got := envelope.Result.Headers["User-Agent"]; len(got) != 1 || got[0] != "Antigravity/0.34.0" {
		t.Fatalf("User-Agent = %#v, want Antigravity/0.34.0", got)
	}
	if got := envelope.Result.Headers["X-Msh-Platform"]; len(got) != 1 || got[0] != "Antigravity" {
		t.Fatalf("X-Msh-Platform = %#v, want Antigravity", got)
	}
	if _, exists := envelope.Result.Headers["Authorization"]; exists {
		t.Fatalf("Authorization must never be rewritten; headers=%#v", envelope.Result.Headers)
	}
	if _, exists := envelope.Result.Headers["Anthropic-Version"]; exists {
		t.Fatalf("Anthropic-Version must be untouched; headers=%#v", envelope.Result.Headers)
	}
}

func TestInterceptBeforeHeaderMappingsNeverApplyToBody(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(filterConfig{
		Mode:               filterModeRewrite,
		UseDefaultKeywords: false,
		RewriteHeaders:     true,
		HeaderMappings:     []rewriteMapping{{Match: "pi", Replacement: "Antigravity"}},
	})

	request := interceptRequestWithHeaders(t, `{"system":"You are pi, an assistant."}`, map[string][]string{
		"originator": {"pi"},
	})

	raw, code := handlePluginCall("request.intercept_before", request)
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Body    string              `json:"Body"`
			Headers map[string][]string `json:"Headers"`
		} `json:"result"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if !envelope.OK {
		t.Fatalf("ok = false, want true")
	}
	if envelope.Result.Body != "" {
		body, _ := base64.StdEncoding.DecodeString(envelope.Result.Body)
		t.Fatalf("header-only mapping %q leaked into body: %s", "pi", body)
	}
	if got := envelope.Result.Headers["originator"]; len(got) != 1 || got[0] != "Antigravity" {
		t.Fatalf("originator = %#v, want Antigravity", got)
	}
}

func TestInterceptBeforeSkipsHeadersWhenDisabled(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(filterConfig{
		Mode:               filterModeRewrite,
		UseDefaultKeywords: true,
		RewriteHeaders:     false,
		HeaderMappings:     []rewriteMapping{{Match: "kimi-code-cli", Replacement: "Antigravity"}},
	})

	request := interceptRequestWithHeaders(t, `{"system":"plain"}`, map[string][]string{
		"User-Agent": {"kimi-code-cli/0.34.0"},
	})

	raw, code := handlePluginCall("request.intercept_before", request)
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Body    string              `json:"Body"`
			Headers map[string][]string `json:"Headers"`
		} `json:"result"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if !envelope.OK {
		t.Fatalf("ok = false, want true")
	}
	if len(envelope.Result.Headers) != 0 || envelope.Result.Body != "" {
		t.Fatalf("response = %s, want no modifications with rewrite_headers=false", raw)
	}
}

// --- config parsing ---

func TestParseFilterConfigYAMLAcceptsHeaderOptions(t *testing.T) {
	cfg, err := parseFilterConfigYAML([]byte(`
mode: rewrite
rewrite_headers: true
header_mappings:
  pi: Antigravity
  kimi-code-cli: Antigravity
`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if !cfg.RewriteHeaders {
		t.Fatal("RewriteHeaders = false, want true")
	}
	if len(cfg.HeaderMappings) != 2 {
		t.Fatalf("HeaderMappings = %#v, want 2 entries", cfg.HeaderMappings)
	}
}

func TestParseFilterConfigYAMLDefaultsRewriteHeadersOn(t *testing.T) {
	cfg, err := parseFilterConfigYAML([]byte("mode: rewrite\n"))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if !cfg.RewriteHeaders {
		t.Fatal("RewriteHeaders = false, want default true")
	}
}

func TestParseFilterConfigYAMLRejectsInvalidHeaderMappings(t *testing.T) {
	if _, err := parseFilterConfigYAML([]byte("header_mappings:\n  - nested: invalid\n")); err == nil {
		t.Fatal("expected error for invalid header_mappings, got nil")
	}
}

// interceptRequestWithHeaders builds a request.intercept_before payload that
// carries headers alongside the body, mirroring pluginapi.RequestInterceptRequest.
func interceptRequestWithHeaders(t *testing.T, body string, headers map[string][]string) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"SourceFormat":   "openai",
		"ToFormat":       "",
		"Model":          "antigravity/test",
		"RequestedModel": "antigravity/test",
		"Headers":        headers,
		"Body":           []byte(body),
	})
	if err != nil {
		t.Fatalf("marshal request intercept request: %v", err)
	}
	return raw
}
