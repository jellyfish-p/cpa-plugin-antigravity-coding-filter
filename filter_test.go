package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestRewriteRequestReplacesDefaultSystemKeywords(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "string system mentions opencode",
			body: `{"system":"You are OpenCode, an AI coding tool."}`,
			want: "You are Antigravity, an AI coding tool.",
		},
		{
			name: "array system mentions claude code",
			body: `{"system":[{"type":"text","text":"Run as Claude Code."}]}`,
			want: "Run as Antigravity.",
		},
		{
			name: "case insensitive codex",
			body: `{"system":"route this CODEX session"}`,
			want: "route this Antigravity session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rewritten := rewriteRequestBody([]byte(tt.body))
			if !rewritten {
				t.Fatalf("rewritten = false, want true")
			}
			if !containsSystemText(t, got, tt.want) {
				t.Fatalf("rewritten body = %s, want system text %q", got, tt.want)
			}
		})
	}
}

func TestBuiltInKeywordPresetCoversMainstreamCodingToolsAndAgents(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(defaultFilterConfig())

	for _, mapping := range defaultRewriteMappings {
		keyword := mapping.Match
		t.Run(keyword, func(t *testing.T) {
			decision := classifyRequest([]byte(`{"system":"You are ` + keyword + `."}`))
			if !decision.Blocked {
				t.Fatalf("%q was not detected by built-in preset", keyword)
			}
			got, rewritten := rewriteRequestBody([]byte(`{"system":"You are ` + keyword + `."}`))
			if !rewritten || !strings.Contains(string(got), "You are Antigravity.") {
				t.Fatalf("%q rewrite = %s, changed=%v", keyword, got, rewritten)
			}
		})
	}
}

func TestDefaultFilterModeBlocksAndRewriteMustBeSelected(t *testing.T) {
	cfg, err := parseFilterConfigYAML(nil)
	if err != nil {
		t.Fatalf("parse default config: %v", err)
	}
	if cfg.Mode != filterModeBlock {
		t.Fatalf("default mode = %q, want block", cfg.Mode)
	}

	cfg, err = parseFilterConfigYAML([]byte("mode: rewrite\n"))
	if err != nil {
		t.Fatalf("parse rewrite config: %v", err)
	}
	if cfg.Mode != filterModeRewrite {
		t.Fatalf("configured mode = %q, want rewrite", cfg.Mode)
	}
}

func TestLongerBuiltInNamesAreRewrittenBeforeShortAliases(t *testing.T) {
	got, rewritten := rewriteRequestBody([]byte(`{"system":"Run GitHub Copilot CLI and OpenAI Codex."}`))
	if !rewritten {
		t.Fatal("rewritten = false, want true")
	}
	if string(got) != `{"system":"Run Antigravity and Antigravity."}` {
		t.Fatalf("body = %s, want complete product names replaced once", got)
	}
}

func TestRewriteRequestIgnoresKeywordsOutsideSystem(t *testing.T) {
	body := []byte(`{
		"messages":[{"role":"user","content":"please compare OpenCode and Codex"}],
		"input":"Claude Code is mentioned by the user"
	}`)
	got, rewritten := rewriteRequestBody(body)
	if rewritten {
		t.Fatalf("rewritten = true, want false; body=%s", got)
	}
}

func TestRewriteRequestAllowsCleanInvalidAndStructuralBodies(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "clean json",
			body: `{"system":"You are Antigravity.","messages":[{"role":"user","content":"hello"}]}`,
		},
		{
			name: "invalid json",
			body: `{`,
		},
		{
			name: "multiple JSON values",
			body: `{"system":"You are Codex."} {}`,
		},
		{
			name: "empty body",
			body: ``,
		},
		{
			name: "prompt cache key",
			body: `{"prompt_cache_key":"session-cache","system":"plain"}`,
		},
		{
			name: "metadata user id",
			body: `{"metadata":{"user_id":"user-123"},"system":"plain"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rewritten := rewriteRequestBody([]byte(tt.body))
			if rewritten {
				t.Fatalf("rewritten = true, want false; body=%s", got)
			}
		})
	}
}

func TestRewriteRequestCoversSupportedProtocolPromptFields(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "responses instructions",
			body: `{"instructions":"You are Codex.","input":[{"role":"user","content":"Keep Codex here."}]}`,
			want: `{"instructions":"You are Antigravity.","input":[{"role":"user","content":"Keep Codex here."}]}`,
		},
		{
			name: "responses developer input text",
			body: `{"input":[{"role":"developer","content":[{"type":"input_text","text":"You are Codex."}]},{"role":"user","content":[{"type":"input_text","text":"Keep Codex here."}]}]}`,
			want: `{"input":[{"role":"developer","content":[{"type":"input_text","text":"You are Antigravity."}]},{"role":"user","content":[{"type":"input_text","text":"Keep Codex here."}]}]}`,
		},
		{
			name: "chat system and developer text blocks",
			body: `{"messages":[{"role":"system","content":[{"type":"text","text":"You are Codex."}]},{"role":"developer","content":"Use Codex rules."},{"role":"user","content":"Keep Codex here."}]}`,
			want: `{"messages":[{"role":"system","content":[{"type":"text","text":"You are Antigravity."}]},{"role":"developer","content":"Use Antigravity rules."},{"role":"user","content":"Keep Codex here."}]}`,
		},
		{
			name: "gemini camel case",
			body: `{"systemInstruction":{"parts":[{"text":"You are Codex."},{"inlineData":{"data":"Codex"}}]}}`,
			want: `{"systemInstruction":{"parts":[{"text":"You are Antigravity."},{"inlineData":{"data":"Codex"}}]}}`,
		},
		{
			name: "gemini snake case wrapper",
			body: `{"request":{"system_instruction":{"parts":[{"text":"You are Codex."}]},"contents":[{"role":"user","parts":[{"text":"Keep Codex here."}]}]}}`,
			want: `{"request":{"system_instruction":{"parts":[{"text":"You are Antigravity."}]},"contents":[{"role":"user","parts":[{"text":"Keep Codex here."}]}]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rewritten := rewriteRequestBody([]byte(tt.body))
			if !rewritten {
				t.Fatalf("rewritten = false, want true")
			}
			var gotValue, wantValue any
			if err := json.Unmarshal(got, &gotValue); err != nil {
				t.Fatalf("decode rewritten body: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.want), &wantValue); err != nil {
				t.Fatalf("decode expected body: %v", err)
			}
			if !reflect.DeepEqual(gotValue, wantValue) {
				t.Fatalf("body = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestRewriteRequestIgnoresIneligibleProtocolText(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"Codex"},{"role":"assistant","content":"Codex"},{"role":"tool","content":"Codex"}],"tools":[{"type":"function","function":{"description":"Codex"}}],"metadata":{"name":"Codex"}}`)
	if got, rewritten := rewriteRequestBody(body); rewritten {
		t.Fatalf("rewritten = true, want false; body=%s", got)
	}
	if decision := classifyRequest(body); decision.Blocked {
		t.Fatalf("decision = %#v, want clean request", decision)
	}
}

func TestRewriteScopeSelectsTextBlocksIndependently(t *testing.T) {
	const request = `{
		"instructions":"system Sample product", "input":"user Sample product",
		"messages":[
			{"role":"developer","content":"system Sample product"},
			{"role":"user","content":[{"type":"text","text":"user Sample product"}]},
			{"role":"assistant","content":[{"type":"output_text","text":"assistant Sample product"}],"tool_calls":[{"id":"Sample product","function":{"name":"Sample product","arguments":"Sample product"}}]},
			{"role":"tool","tool_call_id":"Sample product","content":"tool_results Sample product"}
		],
		"tools":[{"type":"function","function":{"description":"tool_descriptions Sample product"}}],
		"metadata":{"tag":"Sample product","n":9007199254740993}
	}`
	tests := []struct {
		name   string
		config string
		blocks []string
	}{
		{"omitted", "", []string{"system"}},
		{"empty", "scope: []", nil},
		{"user and assistant", "scope: [' User ', assistant, user]", []string{"user", "assistant"}},
		{"tool descriptions only", "scope: [tool_descriptions]", []string{"tool_descriptions"}},
		{"all", "scope: [system, user, assistant, tool_results, tool_descriptions]", []string{"system", "user", "assistant", "tool_results", "tool_descriptions"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseFilterConfigYAML([]byte("mode: rewrite\n" + tt.config + "\ncustom_mappings: {Sample product: replacement}"))
			if err != nil {
				t.Fatal(err)
			}
			got, rewritten := rewriteRequestBodyWithConfig([]byte(request), cfg)
			if rewritten != (len(tt.blocks) > 0) {
				t.Fatalf("rewritten = %v for config %q", rewritten, tt.config)
			}
			if decision := classifyRequestWithConfig([]byte(request), cfg); decision.Blocked != rewritten {
				t.Fatalf("blocked = %v, want %v for config %q", decision.Blocked, rewritten, tt.config)
			}
			if !rewritten {
				return
			}
			want := request
			for _, block := range tt.blocks {
				want = strings.ReplaceAll(want, block+" Sample product", block+" replacement")
			}
			assertRequestJSON(t, got, want)
			if !strings.Contains(string(got), `"n":9007199254740993`) {
				t.Fatalf("large number changed in %s", got)
			}
		})
	}
}

func TestRewriteScopeCoversProtocolToolDescriptions(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "Chat Completions tools",
			body: `{"tools":[{"type":"function","function":{"name":"Codex","description":"description Codex","parameters":{"type":"object","properties":{"query":{"type":"string","description":"parameter Codex"}}}}}],"tool_choice":{"type":"function","function":{"name":"Codex"}},"messages":[{"role":"assistant","tool_calls":[{"id":"Codex","type":"function","function":{"name":"Codex","arguments":"{}"}}]}]}`,
		},
		{
			name: "Chat Completions legacy functions",
			body: `{"functions":[{"name":"Codex","description":"description Codex","parameters":{"type":"object","description":"parameter Codex"}}],"function_call":{"name":"Codex"}}`,
		},
		{
			name: "Responses tools",
			body: `{"tools":[{"type":"function","name":"Codex","description":"description Codex","parameters":{"type":"object","description":"parameter Codex"}},{"type":"custom","name":"tool","description":"description Codex"}],"input":[{"type":"function_call_output","call_id":"Codex","output":"result Codex"}]}`,
		},
		{
			name: "Anthropic tools",
			body: `{"tools":[{"name":"Codex","description":"description Codex","input_schema":{"type":"object","description":"parameter Codex"}}],"metadata":{"description":"metadata Codex","tools":[{"description":"metadata Codex"}]}}`,
		},
		{
			name: "Gemini tools in Antigravity request wrapper",
			body: `{"request":{"tools":[{"functionDeclarations":[{"name":"Codex","description":"description Codex","parameters":{"type":"object","description":"parameter Codex"}}]},{"function_declarations":[{"name":"Codex","description":"description Codex"}]},{"googleSearch":{}}]}}`,
		},
	}
	cfg, err := parseFilterConfigYAML([]byte("mode: rewrite\nscope: [tool_descriptions]"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rewritten := rewriteRequestBodyWithConfig([]byte(tt.body), cfg)
			if !rewritten {
				t.Fatal("tool description was not rewritten")
			}
			assertRequestJSON(t, got, strings.ReplaceAll(tt.body, "description Codex", "description Antigravity"))
			if decision := classifyRequestWithConfig([]byte(tt.body), cfg); !decision.Blocked || decision.Signal != "tool_descriptions.keyword" {
				t.Fatalf("tool description should trigger a block: %#v", decision)
			}
		})
	}
}

func TestRewriteScopeCoversProtocolToolResultText(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "Anthropic mixed user blocks and tool results",
			body: `{"messages":[{"role":"user","content":[{"type":"text","text":"user Codex"},{"type":"tool_result","tool_use_id":"Codex","content":[{"type":"text","text":"result Codex"},{"type":"image","source":{"data":"Codex"}}]},{"type":"tool_result","tool_use_id":"Codex","content":"result Codex"}]}]}`,
		},
		{
			name: "Responses function outputs",
			body: `{"input":[{"type":"function_call","call_id":"Codex","name":"Codex","arguments":"Codex"},{"type":"function_call_output","call_id":"Codex","output":"result Codex"},{"type":"custom_tool_call_output","call_id":"Codex","output":[{"type":"input_text","text":"result Codex"}]}]}`,
		},
		{
			name: "wrapped Gemini function responses",
			body: `{"request":{"contents":[{"role":"user","parts":[{"text":"user Codex"},{"functionResponse":{"name":"Codex","id":"Codex","response":{"result":"result Codex","structured":{"tag":"Codex"}},"parts":[{"text":"result Codex"},{"inlineData":{"data":"Codex"}}]}}]},{"role":"function","parts":[{"function_response":{"name":"Codex","id":"Codex","response":{"output":[{"type":"text","text":"result Codex"}]}}},{"text":"result Codex"}]},{"role":"model","parts":[{"functionCall":{"name":"Codex","args":{"query":"Codex"}}}]}]}}`,
		},
	}
	cfg := defaultFilterConfig()
	cfg.Scope = []scopeBlock{scopeToolResults}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rewritten := rewriteRequestBodyWithConfig([]byte(tt.body), cfg)
			if !rewritten {
				t.Fatal("tool result text was not rewritten")
			}
			assertRequestJSON(t, got, strings.ReplaceAll(tt.body, "result Codex", "result Antigravity"))
			if decision := classifyRequestWithConfig([]byte(tt.body), cfg); !decision.Blocked || decision.Signal != "tool_results.keyword" {
				t.Fatalf("tool result should trigger a block: %#v", decision)
			}
		})
	}
}

func TestRewriteScopeCoversGeminiMessageText(t *testing.T) {
	const body = `{"request":{"contents":[{"role":"user","parts":[{"text":"Codex user"},{"functionResponse":{"response":{"result":"Codex result"}}}]},{"parts":[{"text":"Codex user"}]},{"role":"model","parts":[{"text":"Codex model"},{"thought":true,"text":"Codex thought"}]}]}}`
	cfg := defaultFilterConfig()
	cfg.Scope = []scopeBlock{scopeUser, scopeAssistant}
	got, rewritten := rewriteRequestBodyWithConfig([]byte(body), cfg)
	if !rewritten {
		t.Fatal("Gemini user and model text was not rewritten")
	}
	want := strings.NewReplacer("Codex user", "Antigravity user", "Codex model", "Antigravity model").Replace(body)
	assertRequestJSON(t, got, want)
}

func TestRewriteScopePreservesUnicodeBeforeMatches(t *testing.T) {
	cfg := defaultFilterConfig()
	cfg.Scope = []scopeBlock{scopeUser}
	body := []byte(`{"input":"Kelvin mentioned Codex near https://example.test/Codex."}`)
	got, rewritten := rewriteRequestBodyWithConfig(body, cfg)
	if !rewritten {
		t.Fatal("user message text was not rewritten")
	}
	assertRequestJSON(t, got, `{"input":"Kelvin mentioned Antigravity near https://example.test/Codex."}`)
}

func TestSystemPromptPresetsDetectDeepSeekHarnessAndPi(t *testing.T) {
	tests := []struct {
		prompt string
		want   string
	}{
		{"You are an AI agent powered by DeepSeek Harness.", "You are an AI agent powered by Antigravity."},
		{"You are an expert coding assistant operating inside pi, a coding agent harness.", "You are an expert coding assistant operating inside Antigravity, a coding agent harness."},
		{"You are a coding assistant running inside PI, a coding agent harness.", "You are a coding assistant running inside Antigravity, a coding agent harness."},
	}
	cfg := defaultFilterConfig()
	cfg.Scope = []scopeBlock{scopeSystem, scopeUser}
	for _, tt := range tests {
		t.Run(tt.prompt, func(t *testing.T) {
			body := `{"system":"` + tt.prompt + `","input":"` + tt.prompt + `"}`
			if decision := classifyRequestWithConfig([]byte(body), cfg); !decision.Blocked {
				t.Fatalf("system prompt was not detected: %s", body)
			}
			got, rewritten := rewriteRequestBodyWithConfig([]byte(body), cfg)
			if !rewritten {
				t.Fatalf("system prompt was not rewritten: %s", body)
			}
			assertRequestJSON(t, got, `{"system":"`+tt.want+`","input":"`+tt.prompt+`"}`)
		})
	}
	for _, body := range []string{`{"system":"Describe pi, dsh, and DeepSeek Harness."}`, `{"input":"You are an AI agent powered by DeepSeek Harness."}`} {
		if decision := classifyRequestWithConfig([]byte(body), cfg); decision.Blocked {
			t.Fatalf("unexpected preset match in %s", body)
		}
		if got, rewritten := rewriteRequestBodyWithConfig([]byte(body), cfg); rewritten {
			t.Fatalf("unexpected rewrite = %s", got)
		}
	}
	cfg.UseDefaultKeywords = false
	body := []byte(`{"system":"You are an AI agent powered by DeepSeek Harness."}`)
	if decision := classifyRequestWithConfig(body, cfg); decision.Blocked {
		t.Fatal("preset match with built-in keywords disabled")
	}
	if got, rewritten := rewriteRequestBodyWithConfig(body, cfg); rewritten {
		t.Fatalf("preset rewrite = %s with built-in keywords disabled", got)
	}
}

func assertRequestJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("decode rewritten body: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("decode expected body: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("body = %s, want %s", got, want)
	}
}

func TestRewriteRequestPreservesTechnicalReferencesAndRewritesEmphasizedNames(t *testing.T) {
	body := []byte(`{"system":"mcp__playwright__move-cursor mcp\\_\\_playwright\\_\\_move-cursor CODEX.md __CODEX.md__. .codex/config https://example.test/Codex C:\\work\\Codex \\\\server\\share\\Codex /workspace/Codex docs\\Codex **Codex** __Codex__ \\*\\*Codex\\*\\* Continue.dev Refact.ai SWE-agent"}`)
	got, rewritten := rewriteRequestBody(body)
	if !rewritten {
		t.Fatal("rewritten = false, want true for eligible prose")
	}
	var decoded map[string]string
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	want := `mcp__playwright__move-cursor mcp\_\_playwright\_\_move-cursor CODEX.md __CODEX.md__. .codex/config https://example.test/Codex C:\work\Codex \\server\share\Codex /workspace/Codex docs\Codex **Antigravity** __Antigravity__ \*\*Antigravity\*\* Antigravity Antigravity Antigravity`
	if decoded["system"] != want {
		t.Fatalf("system = %q, want %q", decoded["system"], want)
	}
}

func TestClassifyRequestUsesSameEligiblePromptFieldsAsRewrite(t *testing.T) {
	defer restoreDefaultFilterConfig(t)
	applyFilterConfig(defaultFilterConfig())

	tests := []struct {
		name string
		body string
		want bool
	}{
		{name: "responses developer", body: `{"input":[{"role":"developer","content":[{"type":"input_text","text":"Codex"}]}]}`, want: true},
		{name: "assistant block type under developer role", body: `{"input":[{"role":"developer","content":[{"type":"output_text","text":"Codex"}]}]}`, want: false},
		{name: "chat user", body: `{"messages":[{"role":"user","content":"Codex"}]}`, want: false},
		{name: "gemini system", body: `{"systemInstruction":{"parts":[{"text":"Codex"}]}}`, want: true},
		{name: "technical file", body: `{"system":"CODEX.md"}`, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyRequest([]byte(tt.body)).Blocked; got != tt.want {
				t.Fatalf("blocked = %v, want %v", got, tt.want)
			}
		})
	}
}

func containsSystemText(t *testing.T, body []byte, want string) bool {
	t.Helper()

	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		t.Fatalf("decode rewritten body: %v", err)
	}

	found := false
	walkJSON(root, func(path []string, value any) bool {
		if len(path) == 0 || path[len(path)-1] != "system" {
			return true
		}
		found = strings.Contains(collectText(value), want)
		return !found
	})
	return found
}
