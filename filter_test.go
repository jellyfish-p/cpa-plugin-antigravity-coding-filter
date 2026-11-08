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
