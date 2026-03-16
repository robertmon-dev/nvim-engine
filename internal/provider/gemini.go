package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

	result, err := sendRequest[geminiResponse](req)
	if err != nil {
		return "", err
	}

	if len(result.Candidates) == 0 {
		return "", fmt.Errorf("gemini returned no candidates")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

func (o *GeminiProvider) IsReady() bool {
	return o.APIKey != "" && o.URL != ""
}
