package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"nvim-engine/internal/provider/p_error"
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

func (g *GeminiProvider) Generate(ctx context.Context, system, user string) (string, error) {
	if !g.IsReady() {
		return "", p_error.NewConfigError(string(Gemini))
	}

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

	endpoint := fmt.Sprintf("%s/%s:generateContent?key=%s", g.URL, g.Model, g.APIKey)
	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	return performRequest(ctx, Gemini, req, func(res geminiResponse) string {
		if len(res.Candidates) > 0 && len(res.Candidates[0].Content.Parts) > 0 {
			return res.Candidates[0].Content.Parts[0].Text
		}
		return ""
	})
}

func (o *GeminiProvider) IsReady() bool {
	return o.APIKey != "" && o.URL != ""
}
