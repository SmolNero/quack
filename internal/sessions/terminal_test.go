package sessions

import "testing"

func TestKittyTerminalTitlesFromJSON(t *testing.T) {
	data := []byte(`[
		{"tabs":[{"title":"fallback","windows":[
			{"pid":101,"title":"OC | Example Course","foreground_processes":[{"pid":102}]},
			{"pid":201,"title":"OC | Example Builder","foreground_processes":[{"pid":202}]}
		]}]}
	]`)

	titles := kittyTerminalTitlesFromJSON(data)
	if got := titles[102]; got != "Example Course" {
		t.Fatalf("course title = %q", got)
	}
	if got := titles[202]; got != "Example Builder" {
		t.Fatalf("builder title = %q", got)
	}
}

func TestNormalizeTerminalTitle(t *testing.T) {
	for input, want := range map[string]string{
		"OC | Example Course":        "Example Course",
		"OpenCode | Example Builder": "Example Builder",
		"Custom designated session":  "Custom designated session",
	} {
		if got := normalizeTerminalTitle(input); got != want {
			t.Errorf("normalizeTerminalTitle(%q) = %q, want %q", input, got, want)
		}
	}
}
