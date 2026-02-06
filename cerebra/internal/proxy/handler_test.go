package proxy

import (
	"testing"
)

func TestExtractModel(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		provider Provider
		want     string
	}{
		{
			name:     "OpenAI model",
			body:     `{"model": "gpt-4o", "messages": [{"role": "user", "content": "hello"}]}`,
			provider: ProviderOpenAI,
			want:     "gpt-4o",
		},
		{
			name:     "Anthropic model",
			body:     `{"model": "claude-sonnet-4-20250514", "messages": [{"role": "user", "content": "hello"}]}`,
			provider: ProviderAnthropic,
			want:     "claude-sonnet-4-20250514",
		},
		{
			name:     "Gemini model",
			body:     `{"model": "gemini-2.0-flash", "contents": []}`,
			provider: ProviderGemini,
			want:     "gemini-2.0-flash",
		},
		{
			name:     "no model field",
			body:     `{"messages": []}`,
			provider: ProviderOpenAI,
			want:     "unknown",
		},
		{
			name:     "invalid JSON",
			body:     `not json`,
			provider: ProviderOpenAI,
			want:     "unknown",
		},
		{
			name:     "empty body",
			body:     `{}`,
			provider: ProviderOpenAI,
			want:     "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractModel([]byte(tt.body), tt.provider)
			if got != tt.want {
				t.Errorf("extractModel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractTokenUsage(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		provider   Provider
		wantInput  int64
		wantOutput int64
	}{
		{
			name: "OpenAI usage",
			body: `{
				"usage": {
					"prompt_tokens": 100,
					"completion_tokens": 50,
					"total_tokens": 150
				}
			}`,
			provider:   ProviderOpenAI,
			wantInput:  100,
			wantOutput: 50,
		},
		{
			name: "Anthropic usage",
			body: `{
				"usage": {
					"input_tokens": 200,
					"output_tokens": 75
				}
			}`,
			provider:   ProviderAnthropic,
			wantInput:  200,
			wantOutput: 75,
		},
		{
			name: "Gemini usage",
			body: `{
				"usageMetadata": {
					"promptTokenCount": 300,
					"candidatesTokenCount": 120
				}
			}`,
			provider:   ProviderGemini,
			wantInput:  300,
			wantOutput: 120,
		},
		{
			name:       "no usage data",
			body:       `{"choices": []}`,
			provider:   ProviderOpenAI,
			wantInput:  0,
			wantOutput: 0,
		},
		{
			name:       "invalid JSON",
			body:       `bad`,
			provider:   ProviderOpenAI,
			wantInput:  0,
			wantOutput: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, output := extractTokenUsage([]byte(tt.body), tt.provider)
			if input != tt.wantInput {
				t.Errorf("input tokens = %d, want %d", input, tt.wantInput)
			}
			if output != tt.wantOutput {
				t.Errorf("output tokens = %d, want %d", output, tt.wantOutput)
			}
		})
	}
}

func TestBuildUpstreamURL(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		path     string
		want     string
	}{
		{
			name:     "OpenAI chat completions",
			provider: ProviderOpenAI,
			path:     "/v1/proxy/openai/v1/chat/completions",
			want:     "https://api.openai.com/v1/chat/completions",
		},
		{
			name:     "Anthropic messages",
			provider: ProviderAnthropic,
			path:     "/v1/proxy/anthropic/v1/messages",
			want:     "https://api.anthropic.com/v1/messages",
		},
		{
			name:     "Gemini generate",
			provider: ProviderGemini,
			path:     "/v1/proxy/gemini/v1/models/gemini-pro:generateContent",
			want:     "https://generativelanguage.googleapis.com/v1/models/gemini-pro:generateContent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildUpstreamURL(tt.provider, tt.path)
			if got != tt.want {
				t.Errorf("buildUpstreamURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want float64
	}{
		{"float64", float64(42.5), 42.5},
		{"int", int(10), 10.0},
		{"int64", int64(100), 100.0},
		{"string", "hello", 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toFloat(tt.val)
			if got != tt.want {
				t.Errorf("toFloat(%v) = %f, want %f", tt.val, got, tt.want)
			}
		})
	}
}
