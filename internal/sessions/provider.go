package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ActiveSession struct {
	PID       int
	SessionID string
	Title     string
	Directory string
	StartedAt time.Time
	UpdatedAt time.Time
	Command   string
}

type Provider interface {
	ListActive(ctx context.Context) ([]ActiveSession, error)
	Cancel(ctx context.Context, session ActiveSession) error
}

type OpenCodeProvider struct {
	binary string
}

func NewOpenCodeProvider(binary string) *OpenCodeProvider {
	if strings.TrimSpace(binary) == "" {
		binary = "opencode"
	}
	return &OpenCodeProvider{binary: binary}
}

func (p *OpenCodeProvider) ListActive(ctx context.Context) ([]ActiveSession, error) {
	sessionsByDir, err := p.fetchSessionsByDirectory(ctx)
	if err != nil {
		return nil, err
	}

	procs, err := p.fetchOpenCodeProcesses(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	active := make([]ActiveSession, 0, len(procs))
	for _, proc := range procs {
		item := ActiveSession{
			PID:       proc.pid,
			Directory: proc.cwd,
			StartedAt: now.Add(-time.Duration(proc.elapsedSeconds) * time.Second),
			Command:   proc.command,
		}

		if match, ok := sessionsByDir[proc.cwd]; ok {
			item.SessionID = match.ID
			item.Title = match.Title
			if match.Updated > 0 {
				item.UpdatedAt = time.UnixMilli(match.Updated)
			}
		}

		active = append(active, item)
	}

	sort.Slice(active, func(i, j int) bool {
		if !active[i].UpdatedAt.Equal(active[j].UpdatedAt) {
			return active[i].UpdatedAt.After(active[j].UpdatedAt)
		}
		return active[i].StartedAt.After(active[j].StartedAt)
	})

	return active, nil
}

func (p *OpenCodeProvider) Cancel(ctx context.Context, session ActiveSession) error {
	if session.PID <= 0 {
		return errors.New("missing process ID")
	}
	if err := p.validateSessionProcess(ctx, session); err != nil {
		return err
	}

	proc, err := os.FindProcess(session.PID)
	if err != nil {
		return fmt.Errorf("failed to locate process: %w", err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to terminate session process: %w", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		err := proc.Signal(syscall.Signal(0))
		if err != nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := proc.Kill(); err != nil {
		return fmt.Errorf("process did not exit cleanly: %w", err)
	}

	return nil
}

func (p *OpenCodeProvider) validateSessionProcess(ctx context.Context, session ActiveSession) error {
	procs, err := p.fetchOpenCodeProcesses(ctx)
	if err != nil {
		return err
	}

	for _, proc := range procs {
		if proc.pid != session.PID {
			continue
		}

		if session.Command != "" && proc.command != session.Command {
			return fmt.Errorf("PID %d no longer matches the selected session", session.PID)
		}
		if session.Directory != "" && session.Directory != "unknown" && proc.cwd != "unknown" && proc.cwd != session.Directory {
			return fmt.Errorf("PID %d is now running in a different directory", session.PID)
		}

		return nil
	}

	return fmt.Errorf("PID %d is no longer an active OpenCode session", session.PID)
}

type listedSession struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Updated   int64  `json:"updated"`
	Directory string `json:"directory"`
}

func (p *OpenCodeProvider) fetchSessionsByDirectory(ctx context.Context) (map[string]listedSession, error) {
	cmd := exec.CommandContext(ctx, p.binary, "session", "list", "--format", "json", "-n", "250")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions from opencode: %w", err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return map[string]listedSession{}, nil
	}

	var listed []listedSession
	if err := json.Unmarshal([]byte(trimmed), &listed); err != nil {
		return nil, fmt.Errorf("failed to decode opencode session JSON: %w", err)
	}

	byDir := make(map[string]listedSession, len(listed))
	for _, entry := range listed {
		if entry.Directory == "" {
			continue
		}
		existing, ok := byDir[entry.Directory]
		if !ok || entry.Updated > existing.Updated {
			byDir[entry.Directory] = entry
		}
	}

	return byDir, nil
}

type processRecord struct {
	pid            int
	elapsedSeconds int
	command        string
	cwd            string
}

func (p *OpenCodeProvider) fetchOpenCodeProcesses(ctx context.Context) ([]processRecord, error) {
	cmd := exec.CommandContext(ctx, "ps", "-axo", "pid,etime,command")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect processes: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	records := make([]processRecord, 0, 8)
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		elapsed, err := parseElapsed(fields[1])
		if err != nil {
			continue
		}

		command := strings.Join(fields[2:], " ")
		if !looksLikeOpenCodeSession(command) {
			continue
		}

		cwd, err := readWorkingDirectory(ctx, pid)
		if err != nil {
			cwd = "unknown"
		}

		records = append(records, processRecord{
			pid:            pid,
			elapsedSeconds: elapsed,
			command:        command,
			cwd:            cwd,
		})
	}

	return records, nil
}

func parseElapsed(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("empty elapsed time")
	}

	var days int
	timePart := value
	if strings.Contains(value, "-") {
		parts := strings.SplitN(value, "-", 2)
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid elapsed time %q", value)
		}

		parsedDays, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid elapsed day in %q: %w", value, err)
		}
		days = parsedDays
		timePart = parts[1]
	}

	segments := strings.Split(timePart, ":")
	if len(segments) < 2 || len(segments) > 3 {
		return 0, fmt.Errorf("invalid elapsed clock in %q", value)
	}

	toInt := func(s string) (int, error) {
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return v, nil
	}

	var hours, mins, secs int
	if len(segments) == 2 {
		parsedMins, err := toInt(segments[0])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes in %q: %w", value, err)
		}
		parsedSecs, err := toInt(segments[1])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds in %q: %w", value, err)
		}
		mins = parsedMins
		secs = parsedSecs
	} else {
		parsedHours, err := toInt(segments[0])
		if err != nil {
			return 0, fmt.Errorf("invalid hours in %q: %w", value, err)
		}
		parsedMins, err := toInt(segments[1])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes in %q: %w", value, err)
		}
		parsedSecs, err := toInt(segments[2])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds in %q: %w", value, err)
		}
		hours = parsedHours
		mins = parsedMins
		secs = parsedSecs
	}

	total := secs + mins*60 + hours*3600 + days*24*3600
	return total, nil
}

func looksLikeOpenCodeSession(command string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}

	binary := filepath.Base(fields[0])
	if binary != "opencode" {
		return false
	}

	if len(fields) == 1 {
		return true
	}

	arg := fields[1]
	if strings.HasPrefix(arg, "-") {
		return true
	}

	excludedSubcommands := map[string]struct{}{
		"acp":        {},
		"agent":      {},
		"auth":       {},
		"completion": {},
		"db":         {},
		"debug":      {},
		"export":     {},
		"github":     {},
		"import":     {},
		"mcp":        {},
		"models":     {},
		"pr":         {},
		"run":        {},
		"serve":      {},
		"session":    {},
		"stats":      {},
		"uninstall":  {},
		"upgrade":    {},
		"web":        {},
	}
	if _, blocked := excludedSubcommands[arg]; blocked {
		return false
	}

	return true
}

func readWorkingDirectory(ctx context.Context, pid int) (string, error) {
	cmd := exec.CommandContext(ctx, "lsof", "-a", "-d", "cwd", "-p", strconv.Itoa(pid), "-Fn")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") && len(line) > 1 {
			return strings.TrimSpace(line[1:]), nil
		}
	}

	return "", errors.New("cwd not found")
}
