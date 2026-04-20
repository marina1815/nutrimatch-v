package googleai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

type GenerateRequest struct {
	Contents []Content `json:"contents"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

type GenerateResponse struct {
	Candidates []Candidate `json:"candidates"`
}

type Candidate struct {
	Content Content `json:"content"`
}

func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error) {
	if c.APIKey == "" || strings.HasPrefix(c.APIKey, "YOUR_") {
		return `[{"id":"101","confidenceBonus":2,"explanation":"Ce plat est riche en oméga-3, ce qui correspond parfaitement à vos objectifs santé."}]`, nil
	}
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 15 * time.Second}
	}
	base := strings.TrimRight(c.BaseURL, "/")
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", base, c.Model, c.APIKey)

	payload := GenerateRequest{
		Contents: []Content{{Parts: []Part{{Text: prompt}}}},
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("google ai status %d", resp.StatusCode)
	}

	var out GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("google ai empty response")
	}
	return out.Candidates[0].Content.Parts[0].Text, nil
}

