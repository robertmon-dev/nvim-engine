package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"nvim-engine/internal/engine/types"
	"nvim-engine/internal/provider/p_error"
)

type GeminiProvider struct {
	APIKeys []string
	Model   string
	URL     string
	current uint64
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPayload struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
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

func (o *GeminiProvider) IsReady() bool {
	return len(o.APIKeys) > 0 && o.URL != ""
}

func (p *GeminiProvider) getNextKey() string {
	if len(p.APIKeys) == 0 {
		return ""
	}
	idx := atomic.AddUint64(&p.current, 1) - 1
	return p.APIKeys[idx%uint64(len(p.APIKeys))]
}

func (g *GeminiProvider) Generate(ctx context.Context, system, user string) (string, error) {
	if !g.IsReady() {
		return "", p_error.NewConfigError(string(Gemini))
	}

	payload := geminiPayload{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: system}},
		},
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: user}},
			},
		},
	}

	return g.doRequest(ctx, payload)
}

func (g *GeminiProvider) GenerateChat(ctx context.Context, system string, messages []types.Message) (string, error) {
	payload := geminiPayload{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: system}},
		},
	}

	for _, msg := range messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		payload.Contents = append(payload.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: msg.Content}},
		})
	}

	return g.doRequest(ctx, payload)
}

func (g *GeminiProvider) doRequest(ctx context.Context, payload geminiPayload) (string, error) {
	key := g.getNextKey()
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/%s:generateContent?key=%s", g.URL, g.Model, key)
	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	return performRequest(ctx, Gemini, req, func(res geminiResponse) string {
		if len(res.Candidates) > 0 && len(res.Candidates[0].Content.Parts) > 0 {
			return res.Candidates[0].Content.Parts[0].Text
		}
		return ""
	})
}
