package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const usageEndpoint = "https://chatgpt.com/backend-api/wham/usage"

type Window struct {
	UsedPercent float64
	ResetAt     time.Time
	Window      time.Duration
}

func (w Window) RemainingPercent() float64 {
	remaining := 100 - w.UsedPercent
	if remaining < 0 {
		return 0
	}
	if remaining > 100 {
		return 100
	}
	return remaining
}

type Limits struct {
	FiveHour *Window
	Weekly   *Window
}

type Provider interface {
	Limits(ctx context.Context) (Limits, error)
}

type OpenAIProvider struct {
	authPath string
	endpoint string
	client   *http.Client
}

func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		authPath: openCodeAuthPath(),
		endpoint: usageEndpoint,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

type credential struct {
	Type      string `json:"type"`
	Access    string `json:"access"`
	AccountID string `json:"accountId"`
	Expires   int64  `json:"expires"`
}

type usageResponse struct {
	RateLimit struct {
		Primary   *usageWindow `json:"primary_window"`
		Secondary *usageWindow `json:"secondary_window"`
	} `json:"rate_limit"`
}

type usageWindow struct {
	UsedPercent       float64 `json:"used_percent"`
	LimitWindow       int64   `json:"limit_window_seconds"`
	ResetAfterSeconds int64   `json:"reset_after_seconds"`
	ResetAt           int64   `json:"reset_at"`
}

func (p *OpenAIProvider) Limits(ctx context.Context) (Limits, error) {
	cred, err := p.readCredential()
	if err != nil {
		return Limits{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint, nil)
	if err != nil {
		return Limits{}, fmt.Errorf("create usage request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cred.Access)
	req.Header.Set("User-Agent", "quack")
	if cred.AccountID != "" {
		req.Header.Set("ChatGPT-Account-Id", cred.AccountID)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return Limits{}, fmt.Errorf("fetch usage: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Limits{}, fmt.Errorf("fetch usage: %s", resp.Status)
	}

	var payload usageResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return Limits{}, fmt.Errorf("decode usage: %w", err)
	}

	now := time.Now()
	primary := makeWindow(payload.RateLimit.Primary, now)
	secondary := makeWindow(payload.RateLimit.Secondary, now)
	if primary == nil && secondary == nil {
		return Limits{}, errors.New("usage windows are unavailable")
	}

	limits := Limits{}
	switch {
	case primary != nil && secondary != nil && primary.Window <= secondary.Window:
		limits.FiveHour, limits.Weekly = primary, secondary
	case primary != nil && secondary != nil:
		limits.FiveHour, limits.Weekly = secondary, primary
	case primary != nil && primary.Window >= 24*time.Hour:
		limits.Weekly = primary
	case primary != nil:
		limits.FiveHour = primary
	case secondary.Window >= 24*time.Hour:
		limits.Weekly = secondary
	default:
		limits.FiveHour = secondary
	}
	return limits, nil
}

func makeWindow(window *usageWindow, now time.Time) *Window {
	if window == nil {
		return nil
	}

	resetAt := time.Time{}
	if window.ResetAt > 0 {
		resetAt = time.Unix(window.ResetAt, 0)
	} else if window.ResetAfterSeconds > 0 {
		resetAt = now.Add(time.Duration(window.ResetAfterSeconds) * time.Second)
	}
	return &Window{
		UsedPercent: window.UsedPercent,
		ResetAt:     resetAt,
		Window:      time.Duration(window.LimitWindow) * time.Second,
	}
}

func (p *OpenAIProvider) readCredential() (credential, error) {
	data, err := os.ReadFile(p.authPath)
	if err != nil {
		return credential{}, errors.New("OpenAI login not found; run opencode providers login")
	}

	var auth map[string]credential
	if err := json.Unmarshal(data, &auth); err != nil {
		return credential{}, errors.New("OpenCode authentication is invalid")
	}
	cred, ok := auth["openai"]
	if !ok || cred.Type != "oauth" || strings.TrimSpace(cred.Access) == "" {
		return credential{}, errors.New("OpenAI OAuth login required; run opencode providers login")
	}
	if cred.Expires > 0 && cred.Expires <= time.Now().UnixMilli() {
		return credential{}, errors.New("OpenAI login expired; run opencode providers login")
	}
	return cred, nil
}

func openCodeAuthPath() string {
	dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "opencode", "auth.json")
}
