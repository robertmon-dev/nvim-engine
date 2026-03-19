package p_errors

import (
	"fmt"
	"strings"
)

type ErrorCode string

const (
	ErrRateLimit      ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrInvalidRequest ErrorCode = "INVALID_REQUEST"
	ErrUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrInternal       ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrUnknown        ErrorCode = "UNKNOWN_ERROR"
)

var friendlyMessages = map[ErrorCode]string{
	ErrRateLimit:      "You've hit the API rate limit. Please wait a moment before trying again.",
	ErrUnauthorized:   "Authentication failed. Please check if your API key is correct in the config.",
	ErrInvalidRequest: "The request sent to the provider was invalid. Check your prompts or model settings.",
	ErrInternal:       "The AI provider is having some internal issues. Please try again in a few minutes.",
	ErrUnknown:        "An unexpected error occurred while communicating with the AI provider.",
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

	providerName := strings.Title(strings.ToLower(e.Provider))

	return fmt.Sprintf("%s: %s", providerName, base)
}
