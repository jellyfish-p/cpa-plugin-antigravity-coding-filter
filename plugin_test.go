package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestHandlePluginCallRegisterDeclaresRouterAndExecutor(t *testing.T) {
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
	capabilities := result["capabilities"].(map[string]any)
	if capabilities["model_router"] != true {
		t.Fatalf("model_router = %#v, want true", capabilities["model_router"])
	}
	if capabilities["executor"] != true {
		t.Fatalf("executor = %#v, want true", capabilities["executor"])
	}
}

func TestHandlePluginCallModelRouteBlocksCodingSignals(t *testing.T) {
	request := modelRouteRequestJSON(t, `{"system":"You are Codex.","messages":[]}`)

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
	if !envelope.OK {
		t.Fatalf("ok = false, want true")
	}
	if !envelope.Result.Handled {
		t.Fatalf("Handled = false, want true")
	}
	if envelope.Result.TargetKind != "self" {
		t.Fatalf("TargetKind = %q, want self", envelope.Result.TargetKind)
	}
	if !strings.Contains(envelope.Result.Reason, "system.keyword") {
		t.Fatalf("Reason = %q, want system.keyword detail", envelope.Result.Reason)
	}
}

func TestHandlePluginCallModelRoutePassesCleanRequests(t *testing.T) {
	request := modelRouteRequestJSON(t, `{"system":"You are Antigravity.","messages":[]}`)

	raw, code := handlePluginCall("model.route", request)
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
	if !envelope.OK {
		t.Fatalf("ok = false, want true")
	}
	if envelope.Result.Handled {
		t.Fatalf("Handled = true, want false")
	}
}

func TestHandlePluginCallExecutorExecuteReturnsBlockPayload(t *testing.T) {
	raw, code := handlePluginCall("executor.execute", []byte(`{"Model":"antigravity/test"}`))
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Payload string              `json:"Payload"`
			Headers map[string][]string `json:"Headers"`
		} `json:"result"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if !envelope.OK {
		t.Fatalf("ok = false, want true")
	}
	payload, err := base64.StdEncoding.DecodeString(envelope.Result.Payload)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if !strings.Contains(string(payload), "blocked_by_antigravity_coding_filter") {
		t.Fatalf("payload = %s, want blocked error", payload)
	}
	if got := envelope.Result.Headers["content-type"][0]; got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}
}

func TestHandlePluginCallExecutorExecuteStreamReturnsBlockChunk(t *testing.T) {
	raw, code := handlePluginCall("executor.execute_stream", []byte(`{"Model":"antigravity/test"}`))
	if code != 0 {
		t.Fatalf("code = %d, want 0; body=%s", code, raw)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Headers map[string][]string `json:"Headers"`
			Chunks  []struct {
				Payload string `json:"Payload"`
			} `json:"Chunks"`
		} `json:"result"`
	}
	mustUnmarshalJSON(t, raw, &envelope)
	if !envelope.OK {
		t.Fatalf("ok = false, want true")
	}
	if len(envelope.Result.Chunks) != 1 {
		t.Fatalf("len(Chunks) = %d, want 1", len(envelope.Result.Chunks))
	}
	chunk, err := base64.StdEncoding.DecodeString(envelope.Result.Chunks[0].Payload)
	if err != nil {
		t.Fatalf("decode chunk: %v", err)
	}
	if !strings.Contains(string(chunk), "blocked_by_antigravity_coding_filter") {
		t.Fatalf("chunk = %s, want blocked error", chunk)
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

func mustUnmarshalJSON(t *testing.T, raw []byte, out any) {
	t.Helper()
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
}
