package p_error

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestFromResponseClassification(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantCode ErrorCode
		wantMsg  string
	}{
		{"unauthorized", http.StatusUnauthorized, `{"error":{"message":"bad key"}}`, ErrUnauthorized, "bad key"},
		{"forbidden", http.StatusForbidden, `{}`, ErrUnauthorized, ""},
		{"rate limit", http.StatusTooManyRequests, `{"error":{"message":"slow down"}}`, ErrRateLimit, "slow down"},
		{"anthropic overloaded", 529, `{"type":"overloaded_error"}`, ErrInternal, ""},
		{"bad request", http.StatusBadRequest, `{"detail":"nope"}`, ErrInvalidRequest, "nope"},
		{"server error", http.StatusInternalServerError, `{}`, ErrInternal, ""},
		{"unmapped status", http.StatusTeapot, `{}`, ErrUnknown, ""},
		{"safety block on 200", http.StatusOK, `{"candidates":null,"promptFeedback":{"blockReason":"safety"}}`, ErrInvalidRequest, ""},
		{"quota in body wins", http.StatusBadRequest, `{"error":{"code":"insufficient_quota"}}`, ErrRateLimit, "Quota exceeded. Check your billing."},
		{"empty content", http.StatusOK, `{"content":[]}`, ErrInternal, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FromResponse("openai", tt.status, []byte(tt.body))

			var pErr *ProviderError
			if !errors.As(err, &pErr) {
				t.Fatalf("expected *ProviderError, got %T", err)
			}
			if pErr.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", pErr.Code, tt.wantCode)
			}
			if tt.wantMsg != "" && pErr.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", pErr.Message, tt.wantMsg)
			}
			if pErr.Status != tt.status {
				t.Errorf("status = %d, want %d", pErr.Status, tt.status)
			}
		})
	}
}

func TestExtractMessageFallsThroughEmptyNestedObject(t *testing.T) {
	body := []byte(`{"error":{"code":429,"status":"RESOURCE_EXHAUSTED"},"detail":"found me"}`)

	if got := extractMessage(body); got != "found me" {
		t.Errorf("extractMessage() = %q, want %q", got, "found me")
	}
}

func TestExtractMessageNestedHit(t *testing.T) {
	if got := extractMessage([]byte(`{"error":{"message":"deep"}}`)); got != "deep" {
		t.Errorf("extractMessage() = %q, want %q", got, "deep")
	}
}

func TestExtractMessageOnNonJSON(t *testing.T) {
	if got := extractMessage([]byte("<html>502 Bad Gateway</html>")); got != "" {
		t.Errorf("extractMessage() = %q, want empty", got)
	}
}

func TestFriendly(t *testing.T) {
	tests := []struct {
		name string
		err  *ProviderError
		want string
	}{
		{"known code", &ProviderError{Code: ErrRateLimit, Provider: "openai"}, "Openai: You've hit"},
		{"unknown code falls back", &ProviderError{Code: "NOPE", Provider: "gemini"}, "Gemini: An unexpected error"},
		{"empty provider must not panic", &ProviderError{Code: ErrConfig}, "Provider: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Friendly()
			if !strings.HasPrefix(got, tt.want) {
				t.Errorf("Friendly() = %q, want prefix %q", got, tt.want)
			}
		})
	}
}

func TestErrAllProvidersFailedIsMatchable(t *testing.T) {
	wrapped := fmt.Errorf("commit generation failed: %w", ErrAllProvidersFailed)

	if !errors.Is(wrapped, ErrAllProvidersFailed) {
		t.Error("a wrapped ErrAllProvidersFailed must match with errors.Is")
	}
}
