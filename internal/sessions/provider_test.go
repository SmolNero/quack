package sessions

import (
	"strings"
	"testing"
	"time"
)

func TestMatchSessionsFromLog(t *testing.T) {
	started := time.Date(2026, 8, 6, 14, 1, 48, 0, time.UTC)
	procs := []processRecord{
		{pid: 101, startedAt: started},
		{pid: 202, startedAt: started.Add(time.Hour)},
	}
	log := strings.Join([]string{
		`timestamp=2026-08-06T14:01:49.294Z level=INFO run=first message="creating instance" directory="/tmp/Project Name"`,
		`timestamp=2026-08-06T14:04:53.351Z level=INFO run=first message=loop session.id=ses_first123 step=0`,
		`timestamp=2026-08-06T15:01:49.100Z level=INFO run=second message="creating instance" directory=/tmp`,
		`timestamp=2026-08-06T16:01:49.100Z level=INFO run=unrelated message="creating instance" directory=/tmp`,
	}, "\n")

	matches, err := matchSessionsFromLog(strings.NewReader(log), procs)
	if err != nil {
		t.Fatalf("matchSessionsFromLog returned an error: %v", err)
	}
	if got := matches[101]; !got.runFound || got.sessionID != "ses_first123" {
		t.Fatalf("PID 101 match = %#v, want session ses_first123", got)
	}
	if got := matches[101].directory; got != "/tmp/Project Name" {
		t.Fatalf("PID 101 directory = %q, want quoted directory", got)
	}
	if got := matches[202]; !got.runFound || got.sessionID != "" {
		t.Fatalf("PID 202 match = %#v, want a run with no selected session", got)
	}
}

func TestSessionIDFromCommand(t *testing.T) {
	tests := map[string]string{
		"opencode":                         "",
		"opencode -s ses_short123":         "ses_short123",
		"opencode --session ses_long456":   "ses_long456",
		"opencode --session=ses_equals789": "ses_equals789",
		"opencode --session not-valid":     "",
	}
	for command, want := range tests {
		t.Run(command, func(t *testing.T) {
			if got := sessionIDFromCommand(command); got != want {
				t.Fatalf("sessionIDFromCommand(%q) = %q, want %q", command, got, want)
			}
		})
	}
}
