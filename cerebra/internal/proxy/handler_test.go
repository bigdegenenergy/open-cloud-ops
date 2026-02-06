package proxy

import (
	"encoding/json"
	"testing"
)

func TestExtractProviderAndPath_OpenAI(t *testing.T) {
	provider, path, err := extractProviderAndPath("/v1/openai/v1/chat/completions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != ProviderOpenAI {
		t.Errorf("expected provider %q, got %q", ProviderOpenAI, provider)
	}
	if path != "/v1/chat/completions" {
		t.Errorf("expected path %q, got %q", "/v1/chat/completions", path)
	}
}

func TestExtractProviderAndPath_Anthropic(t *testing.T) {
	provider, path, err := extractProviderAndPath("/v1/anthropic/v1/messages")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != ProviderAnthropic {
		t.Errorf("expected provider %q, got %q", ProviderAnthropic, provider)
	}
	if path != "/v1/messages" {
		t.Errorf("expected path %q, got %q", "/v1/messages", path)
	}
}

func TestExtractProviderAndPath_Gemini(t *testing.T) {
	provider, path, err := extractProviderAndPath("/v1/gemini/v1beta/models/gemini-pro:generateContent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != ProviderGemini {
		t.Errorf("expected provider %q, got %q", ProviderGemini, provider)
	}
	if path != "/v1beta/models/gemini-pro:generateContent" {
		t.Errorf("expected path %q, got %q", "/v1beta/models/gemini-pro:generateContent", path)
	}
}

func TestExtractProviderAndPath_InvalidProvider(t *testing.T) {
	_, _, err := extractProviderAndPath("/v1/invalid/chat")
	if err == nil {
		t.Fatal("expected error for invalid provider, got nil")
	}
}

func TestExtractProviderAndPath_InvalidVersion(t *testing.T) {
	_, _, err := extractProviderAndPath("/v2/openai/chat")
	if err == nil {
		t.Fatal("expected error for invalid version, got nil")
	}
}

func TestExtractProviderAndPath_TooShort(t *testing.T) {
	_, _, err := extractProviderAndPath("/v1")
	if err == nil {
		t.Fatal("expected error for short path, got nil")
	}
}

func TestExtractProviderAndPath_RootPath(t *testing.T) {
	provider, path, err := extractProviderAndPath("/v1/openai/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != ProviderOpenAI {
		t.Errorf("expected provider %q, got %q", ProviderOpenAI, provider)
	}
	if path != "/" {
		t.Errorf("expected path %q, got %q", "/", path)
	}
}

func TestExtractModelFromRequest_OpenAI(t *testing.T) {
	body := []byte(`{"model": "gpt-4o", "messages": [{"role": "user", "content": "hello"}]}`)
	model := extractModelFromRequest(ProviderOpenAI, body)
	if model != "gpt-4o" {
		t.Errorf("expected model %q, got %q", "gpt-4o", model)
	}
}

func TestExtractModelFromRequest_Anthropic(t *testing.T) {
	body := []byte(`{"model": "claude-3-5-sonnet-20241022", "max_tokens": 1024, "messages": []}`)
	model := extractModelFromRequest(ProviderAnthropic, body)
	if model != "claude-3-5-sonnet-20241022" {
		t.Errorf("expected model %q, got %q", "claude-3-5-sonnet-20241022", model)
	}
}

func TestExtractModelFromRequest_Gemini(t *testing.T) {
	body := []byte(`{"model": "gemini-1.5-pro", "contents": []}`)
	model := extractModelFromRequest(ProviderGemini, body)
	if model != "gemini-1.5-pro" {
		t.Errorf("expected model %q, got %q", "gemini-1.5-pro", model)
	}
}

func TestExtractModelFromRequest_EmptyBody(t *testing.T) {
	model := extractModelFromRequest(ProviderOpenAI, []byte{})
	if model != "" {
		t.Errorf("expected empty model, got %q", model)
	}
}

func TestExtractModelFromRequest_InvalidJSON(t *testing.T) {
	model := extractModelFromRequest(ProviderOpenAI, []byte("not json"))
	if model != "" {
		t.Errorf("expected empty model, got %q", model)
	}
}

func TestExtractModelFromRequest_NoModel(t *testing.T) {
	body := []byte(`{"messages": [{"role": "user", "content": "hello"}]}`)
	model := extractModelFromRequest(ProviderOpenAI, body)
	if model != "" {
		t.Errorf("expected empty model, got %q", model)
	}
}

func TestExtractTokenUsage_OpenAI(t *testing.T) {
	resp := map[string]interface{}{
		"usage": map[string]interface{}{
			"prompt_tokens":     float64(100),
			"completion_tokens": float64(50),
			"total_tokens":      float64(150),
		},
	}
	body, _ := json.Marshal(resp)

	input, output, total := extractTokenUsage(ProviderOpenAI, body)
	if input != 100 {
		t.Errorf("expected input tokens 100, got %d", input)
	}
	if output != 50 {
		t.Errorf("expected output tokens 50, got %d", output)
	}
	if total != 150 {
		t.Errorf("expected total tokens 150, got %d", total)
	}
}

func TestExtractTokenUsage_Anthropic(t *testing.T) {
	resp := map[string]interface{}{
		"usage": map[string]interface{}{
			"input_tokens":  float64(200),
			"output_tokens": float64(300),
		},
	}
	body, _ := json.Marshal(resp)

	input, output, total := extractTokenUsage(ProviderAnthropic, body)
	if input != 200 {
		t.Errorf("expected input tokens 200, got %d", input)
	}
	if output != 300 {
		t.Errorf("expected output tokens 300, got %d", output)
	}
	if total != 500 {
		t.Errorf("expected total tokens 500, got %d", total)
	}
}

func TestExtractTokenUsage_Gemini(t *testing.T) {
	resp := map[string]interface{}{
		"usageMetadata": map[string]interface{}{
			"promptTokenCount":     float64(150),
			"candidatesTokenCount": float64(250),
			"totalTokenCount":      float64(400),
		},
	}
	body, _ := json.Marshal(resp)

	input, output, total := extractTokenUsage(ProviderGemini, body)
	if input != 150 {
		t.Errorf("expected input tokens 150, got %d", input)
	}
	if output != 250 {
		t.Errorf("expected output tokens 250, got %d", output)
	}
	if total != 400 {
		t.Errorf("expected total tokens 400, got %d", total)
	}
}

func TestExtractTokenUsage_EmptyResponse(t *testing.T) {
	input, output, total := extractTokenUsage(ProviderOpenAI, []byte{})
	if input != 0 || output != 0 || total != 0 {
		t.Errorf("expected all zeros for empty response, got %d/%d/%d", input, output, total)
	}
}

func TestExtractTokenUsage_NoUsageField(t *testing.T) {
	resp := map[string]interface{}{
		"id": "resp-123",
	}
	body, _ := json.Marshal(resp)

	input, output, total := extractTokenUsage(ProviderOpenAI, body)
	if input != 0 || output != 0 || total != 0 {
		t.Errorf("expected all zeros for missing usage, got %d/%d/%d", input, output, total)
	}
}

func TestJsonToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int64
	}{
		{"float64", float64(42), 42},
		{"int64", int64(99), 99},
		{"nil", nil, 0},
		{"string", "hello", 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jsonToInt64(tt.input)
			if result != tt.expected {
				t.Errorf("jsonToInt64(%v) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProviderUpstreamURLs(t *testing.T) {
	tests := []struct {
		provider Provider
		expected string
	}{
		{ProviderOpenAI, "https://api.openai.com"},
		{ProviderAnthropic, "https://api.anthropic.com"},
		{ProviderGemini, "https://generativelanguage.googleapis.com"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			url, ok := providerUpstreamURLs[tt.provider]
			if !ok {
				t.Fatalf("provider %q not found in upstream URLs", tt.provider)
			}
			if url != tt.expected {
				t.Errorf("expected URL %q, got %q", tt.expected, url)
			}
		})
	}
}
