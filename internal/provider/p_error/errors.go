package p_error

import (
	"encoding/json"
	"errors"
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

var ErrAllProvidersFailed = errors.New("all attempted providers failed")

var FriendlyMessages = map[ErrorCode]string{
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
	base, ok := FriendlyMessages[e.Code]
	if !ok {
		base = FriendlyMessages[ErrUnknown]
	}

	return fmt.Sprintf("%s: %s", titleCase(e.Provider), base)
}

func titleCase(s string) string {
	if s == "" {
		return "Provider"
	}

	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
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
	bodyStr := string(body)
	code := codeForStatus(status)
	message := extractMessage(body)

	switch {
	case strings.Contains(bodyStr, "safety") || strings.Contains(bodyStr, `candidates":null`):
		code = ErrInvalidRequest
		if message == "" {
			message = "Content blocked by safety filters."
		}
	case strings.Contains(bodyStr, "overloaded_error"):
		code = ErrInternal
	case strings.Contains(bodyStr, "insufficient_quota"):
		code = ErrRateLimit
		message = "Quota exceeded. Check your billing."
	case strings.Contains(bodyStr, `content":null`), strings.Contains(bodyStr, `content":[]`),
		strings.Contains(bodyStr, "empty content"):
		code = ErrInternal
	}

	if message == "" && bodyStr != "" {
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

func codeForStatus(status int) ErrorCode {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrUnauthorized
	case http.StatusTooManyRequests, 529:
		return ErrRateLimit
	case http.StatusBadRequest, http.StatusNotFound:
		return ErrInvalidRequest
	case http.StatusInternalServerError, http.StatusServiceUnavailable:
		return ErrInternal
	}

	return ErrUnknown
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
			if s := findString(nextMap, keys...); s != "" {
				return s
			}
		}
	}

	return ""
}
