package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/SmolNero/quack/internal/usage"
)

func TestUsageFormatting(t *testing.T) {
	byteTests := map[int64]string{
		0:                  "0 B",
		1024:               "1.0 KB",
		804 * 1024 * 1024:  "804.0 MB",
		1337 * 1024 * 1024: "1.3 GB",
	}
	for value, want := range byteTests {
		if got := formatBytes(value); got != want {
			t.Errorf("formatBytes(%d) = %q, want %q", value, got, want)
		}
	}

	tokenTests := map[int64]string{
		999:       "999",
		4_493_312: "4.5M",
	}
	for value, want := range tokenTests {
		if got := formatTokens(value); got != want {
			t.Errorf("formatTokens(%d) = %q, want %q", value, got, want)
		}
	}
}

func TestUsageView(t *testing.T) {
	m := Model{
		width: 100,
		weekly: usage.Weekly{
			UsedPercent: 89,
			ResetAt:     time.Date(2026, 8, 7, 21, 39, 0, 0, time.Local),
		},
		usageLastSync: time.Date(2026, 8, 6, 8, 15, 0, 0, time.Local),
	}

	view := m.usageView()
	if !strings.Contains(view, "11% remaining") {
		t.Fatalf("usage view does not show the remaining percentage: %q", view)
	}
	if !strings.Contains(view, "resets Fri Aug 7, 9:39 PM") {
		t.Fatalf("usage view does not show the reset time: %q", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if width := lipgloss.Width(line); width > m.width {
			t.Fatalf("usage line width = %d, exceeds terminal width %d", width, m.width)
		}
	}
}
