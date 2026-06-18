package main

import "testing"

func TestClassifyRequestBlocksSystemKeywords(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "string system mentions opencode",
			body: `{"system":"You are OpenCode, an AI coding tool."}`,
		},
		{
			name: "array system mentions claude code",
			body: `{"system":[{"type":"text","text":"Run as Claude Code."}]}`,
		},
		{
			name: "case insensitive codex",
			body: `{"system":"route this CODEX session"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyRequest([]byte(tt.body))
			if !got.Blocked {
				t.Fatalf("Blocked = false, want true")
			}
			if got.Signal == "" {
				t.Fatalf("Signal is empty")
			}
		})
	}
}

func TestClassifyRequestIgnoresKeywordsOutsideSystem(t *testing.T) {
	got := classifyRequest([]byte(`{
		"messages":[{"role":"user","content":"please compare OpenCode and Codex"}],
		"input":"Claude Code is mentioned by the user"
	}`))
	if got.Blocked {
		t.Fatalf("Blocked = true, want false; signal=%q", got.Signal)
	}
}

func TestClassifyRequestBlocksPromptCacheKey(t *testing.T) {
	got := classifyRequest([]byte(`{"prompt_cache_key":"session-cache","system":"plain"}`))
	if !got.Blocked {
		t.Fatalf("Blocked = false, want true")
	}
	if got.Signal != "prompt_cache_key" {
		t.Fatalf("Signal = %q, want prompt_cache_key", got.Signal)
	}
}

func TestClassifyRequestBlocksNestedPromptCacheKey(t *testing.T) {
	got := classifyRequest([]byte(`{"request":{"prompt_cache_key":"session-cache"},"system":"plain"}`))
	if !got.Blocked {
		t.Fatalf("Blocked = false, want true")
	}
	if got.Signal != "prompt_cache_key" {
		t.Fatalf("Signal = %q, want prompt_cache_key", got.Signal)
	}
}

func TestClassifyRequestBlocksMetadataUserID(t *testing.T) {
	got := classifyRequest([]byte(`{"metadata":{"user_id":"user-123"},"system":"plain"}`))
	if !got.Blocked {
		t.Fatalf("Blocked = false, want true")
	}
	if got.Signal != "metadata.user_id" {
		t.Fatalf("Signal = %q, want metadata.user_id", got.Signal)
	}
}

func TestClassifyRequestBlocksNestedMetadataUserID(t *testing.T) {
	got := classifyRequest([]byte(`{"request":{"metadata":{"user_id":"user-123"}},"system":"plain"}`))
	if !got.Blocked {
		t.Fatalf("Blocked = false, want true")
	}
	if got.Signal != "metadata.user_id" {
		t.Fatalf("Signal = %q, want metadata.user_id", got.Signal)
	}
}

func TestClassifyRequestAllowsCleanAndInvalidBodies(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyRequest([]byte(tt.body))
			if got.Blocked {
				t.Fatalf("Blocked = true, want false; signal=%q", got.Signal)
			}
		})
	}
}
