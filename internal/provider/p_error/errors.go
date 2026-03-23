package p_error

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type ErrorCode string

const (
	ErrRateLimit      ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrInvalidRequest ErrorCode = "INVALID_REQUEST"
	ErrUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrInternal       ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrUnknown        ErrorCode = "UNKNOWN_ERROR"
	ErrConfig         ErrorCode = "CONFIGURATION_ERROR"
)

var friendlyMessages = map[ErrorCode]string{
	ErrRateLimit:      "You've hit the API rate limit. Please wait a moment before trying again.",
	ErrUnauthorized:   "Authentication failed. Please check if your API key is correct in the config.",
	ErrInvalidRequest: "The request sent to the provider was invalid. Check your prompts or model settings.",
	ErrInternal:       "The AI provider is having some internal issues. Please try again in a few minutes.",
	ErrUnknown:        "An unexpected error occurred while communicating with the AI provider.",
	ErrConfig:         "Provider is not properly configured. Please check your API key and URL in the configuration.",
}

type ProviderError struct {
	Code     ErrorCode
	Provider string
	Message  string
	Status   int
	Raw      string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("[%s] %d: %s", e.Code, e.Status, e.Message)
}

func (e *ProviderError) Friendly() string {
	base, ok := friendlyMessages[e.Code]
	if !ok {
		base = friendlyMessages[ErrUnknown]
	}

	providerName := strings.ToUpper(e.Provider[:1]) + strings.ToLower(e.Provider[1:])
	return fmt.Sprintf("%s: %s", providerName, base)
}

func NewConfigError(providerName string) error {
	return &ProviderError{
		Code:     ErrConfig,
		Provider: providerName,
		Status:   0,
		Message:  "Missing required configuration (API Key or URL)",
		Raw:      "",
	}
}

func FromResponse(providerName string, status int, body []byte) error {
	code := ErrUnknown
	bodyStr := string(body)

	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		code = ErrUnauthorized
	case http.StatusTooManyRequests, 529:
		code = ErrRateLimit
	case http.StatusBadRequest, http.StatusNotFound:
		code = ErrInvalidRequest
	case http.StatusInternalServerError, http.StatusServiceUnavailable:
		code = ErrInternal
	default:
		if status >= 200 && status < 300 {
			code = ErrUnknown
		}
	}

	message := extractMessage(body)

	if strings.Contains(bodyStr, "safety") || strings.Contains(bodyStr, "candidates\":null") {
		code = ErrInvalidRequest
		if message == "" {
			message = "Content blocked by safety filters."
		}
	} else if strings.Contains(bodyStr, "overloaded_error") {
		code = ErrInternal
	} else if strings.Contains(bodyStr, "insufficient_quota") {
		code = ErrRateLimit
		message = "Quota exceeded. Check your billing."
	} else if strings.Contains(bodyStr, "content\":null") || strings.Contains(bodyStr, "content\":[]") || strings.Contains(bodyStr, "empty content") {
		code = ErrInternal
	}

	if message == "" && len(bodyStr) > 0 {
		message = bodyStr
	}

	return &ProviderError{
		Code:     code,
		Provider: providerName,
		Status:   status,
		Message:  message,
		Raw:      bodyStr,
	}
}

func extractMessage(body []byte) string {
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}

	return findString(data, "message", "error", "text", "detail")
}

func findString(data map[string]any, keys ...string) string {
	for _, k := range keys {
		val, ok := data[k]
		if !ok {
			continue
		}
		if s, ok := val.(string); ok {
			return s
		}

		if nextMap, ok := val.(map[string]any); ok {
			return findString(nextMap, keys...)
		}
	}
	return ""
}
