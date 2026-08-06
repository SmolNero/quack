package usage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWeeklyUsesLongestRateLimitWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer not-a-real-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("ChatGPT-Account-Id"); got != "example-account" {
			t.Errorf("ChatGPT-Account-Id = %q", got)
		}
		_, _ = w.Write([]byte(`{
			"rate_limit": {
				"primary_window": {"used_percent": 20, "limit_window_seconds": 18000, "reset_at": 1786000000},
				"secondary_window": {"used_percent": 89, "limit_window_seconds": 604800, "reset_at": 1786160372}
			}
		}`))
	}))
	defer server.Close()

	authPath := filepath.Join(t.TempDir(), "auth.json")
	auth := `{"openai":{"type":"oauth","access":"not-a-real-token","accountId":"example-account","expires":9999999999999}}`
	if err := os.WriteFile(authPath, []byte(auth), 0o600); err != nil {
		t.Fatal(err)
	}

	provider := &OpenAIProvider{authPath: authPath, endpoint: server.URL, client: server.Client()}
	weekly, err := provider.Weekly(context.Background())
	if err != nil {
		t.Fatalf("Weekly returned an error: %v", err)
	}
	if weekly.UsedPercent != 89 || weekly.RemainingPercent() != 11 {
		t.Fatalf("usage = %.1f%% used / %.1f%% remaining", weekly.UsedPercent, weekly.RemainingPercent())
	}
	if weekly.Window != 7*24*time.Hour {
		t.Fatalf("window = %s, want 7 days", weekly.Window)
	}
	if got := weekly.ResetAt.Unix(); got != 1786160372 {
		t.Fatalf("reset = %d, want 1786160372", got)
	}
}
