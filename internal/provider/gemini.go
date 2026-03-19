package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"nvim-engine/internal/provider/p_errors"
)

type GeminiProvider struct {
	APIKey string
	Model  string
	URL    string
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPayload struct {
	Contents []geminiContent `json:"contents"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type geminiErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func (g *GeminiProvider) mapError(status int, body []byte) error {
	var gemErr geminiErrorResponse
	_ = json.Unmarshal(body, &gemErr)

	code := p_errors.ErrUnknown
	switch status {
	case 401, 403:
		code = p_errors.ErrUnauthorized
	case 429:
		code = p_errors.ErrRateLimit
	case 400:
		code = p_errors.ErrInvalidRequest
	case 500, 503, 504:
		code = p_errors.ErrInternal
	}

	return &p_errors.ProviderError{
		Code:     code,
		Provider: string(Gemini),
		Status:   status,
		Message:  gemErr.Error.Message,
		Raw:      string(body),
	}
}

func (g *GeminiProvider) Generate(ctx context.Context, system, user string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s:generateContent?key=%s", g.URL, g.Model, g.APIKey)

	payload := geminiPayload{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: system + "\n\n" + user},
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	_, body, status, err := sendRequest[geminiResponse](req)
	if err != nil {
		return "", err
	}

	if status < 200 || status >= 300 {
		return "", g.mapError(status, body)
	}

	var result geminiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to decode gemini response: %w", err)
	}

	if len(result.Candidates) == 0 {
		return "", &p_errors.ProviderError{
			Code:     p_errors.ErrInvalidRequest,
			Provider: string(Gemini),
			Status:   status,
			Message:  "Gemini returned no candidates. This usually happens when the content is blocked by safety filters.",
			Raw:      string(body),
		}
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

func (o *GeminiProvider) IsReady() bool {
	return o.APIKey != "" && o.URL != ""
}
