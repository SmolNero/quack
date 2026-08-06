package sessions

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

type kittyOSWindow struct {
	Tabs []kittyTab `json:"tabs"`
}

type kittyTab struct {
	Title   string        `json:"title"`
	Windows []kittyWindow `json:"windows"`
}

type kittyWindow struct {
	PID                 int                `json:"pid"`
	Title               string             `json:"title"`
	ForegroundProcesses []kittyProcessInfo `json:"foreground_processes"`
}

type kittyProcessInfo struct {
	PID int `json:"pid"`
}

// KittyTerminalTitles must run before Bubble Tea starts reading from the TTY.
func KittyTerminalTitles(ctx context.Context) map[int]string {
	if os.Getenv("KITTY_WINDOW_ID") == "" {
		return nil
	}

	cmd := exec.CommandContext(ctx, "kitty", "@", "--use-password", "never", "ls")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return kittyTerminalTitlesFromJSON(out)
}

func kittyTerminalTitlesFromJSON(data []byte) map[int]string {
	var osWindows []kittyOSWindow
	if err := json.Unmarshal(data, &osWindows); err != nil {
		return nil
	}

	titles := make(map[int]string)
	for _, osWindow := range osWindows {
		for _, tab := range osWindow.Tabs {
			for _, window := range tab.Windows {
				title := normalizeTerminalTitle(window.Title)
				if title == "" {
					title = normalizeTerminalTitle(tab.Title)
				}
				if title == "" {
					continue
				}

				if window.PID > 0 {
					titles[window.PID] = title
				}
				for _, process := range window.ForegroundProcesses {
					if process.PID > 0 {
						titles[process.PID] = title
					}
				}
			}
		}
	}
	return titles
}

func normalizeTerminalTitle(title string) string {
	title = strings.TrimSpace(title)
	lower := strings.ToLower(title)
	for _, prefix := range []string{"oc | ", "opencode | "} {
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSpace(title[len(prefix):])
		}
	}
	return title
}
