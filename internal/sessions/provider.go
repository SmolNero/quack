package sessions

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	PID        int
	SessionID  string
	Title      string
	Directory  string
	StartedAt  time.Time
	UpdatedAt  time.Time
	Command    string
	Memory     int64
	CPU        float64
	Input      int64
	Output     int64
	Reasoning  int64
	CacheRead  int64
	CacheWrite int64
}

type Provider interface {
	ListActive(ctx context.Context) ([]ActiveSession, error)
	Cancel(ctx context.Context, session ActiveSession) error
}

type OpenCodeProvider struct {
	binary         string
	terminalTitles map[int]string
}

func (p *OpenCodeProvider) SetTerminalTitles(titles map[int]string) {
	p.terminalTitles = titles
}

func NewOpenCodeProvider(binary string) *OpenCodeProvider {
	if strings.TrimSpace(binary) == "" {
		binary = "opencode"
	}
	return &OpenCodeProvider{binary: binary}
}

func (p *OpenCodeProvider) ListActive(ctx context.Context) ([]ActiveSession, error) {
	procs, err := p.fetchOpenCodeProcesses(ctx)
	if err != nil {
		return nil, err
	}

	matches := p.matchActiveSessions(procs)
	matchedIDs := make([]string, 0, len(matches))
	matchedTitles := make([]string, 0, len(matches))
	for _, match := range matches {
		if match.sessionID != "" {
			matchedIDs = append(matchedIDs, match.sessionID)
		} else if match.title != "" {
			matchedTitles = append(matchedTitles, match.title)
		}
	}

	sessionsByMatch, err := p.fetchSessionsByID(ctx, matchedIDs)
	if err != nil {
		return nil, err
	}
	sessionsByTitle, err := p.fetchSessionsByTitle(ctx, matchedTitles)
	if err != nil {
		return nil, err
	}

	active := make([]ActiveSession, 0, len(procs))
	for _, proc := range procs {
		item := ActiveSession{
			PID:       proc.pid,
			Title:     "Unidentified session",
			Directory: "unknown",
			StartedAt: proc.startedAt,
			Command:   proc.command,
			Memory:    proc.memory,
			CPU:       proc.cpu,
		}

		match := matches[proc.pid]
		if match.directory != "" {
			item.Directory = match.directory
		}
		if match.title != "" {
			item.Title = match.title
		} else if match.runFound && match.sessionID == "" {
			item.Title = "No session selected"
		} else if match.sessionID == "" {
			item.Title = "Unidentified session"
		}

		if details, ok := sessionsByMatch[match.sessionID]; ok {
			applySessionDetails(&item, details)
		} else if details, ok := sessionsByTitle[match.title]; ok {
			applySessionDetails(&item, details)
		}

		active = append(active, item)
	}

	sort.Slice(active, func(i, j int) bool {
		if active[i].Memory != active[j].Memory {
			return active[i].Memory > active[j].Memory
		}
		if !active[i].UpdatedAt.Equal(active[j].UpdatedAt) {
			return active[i].UpdatedAt.After(active[j].UpdatedAt)
		}
		return active[i].StartedAt.After(active[j].StartedAt)
	})

	return active, nil
}

<<<<<<< HEAD
func applySessionDetails(item *ActiveSession, details listedSession) {
	item.SessionID = details.ID
	item.Title = details.Title
	item.Directory = details.Directory
	item.Input = details.Input
	item.Output = details.Output
	item.Reasoning = details.Reasoning
	item.CacheRead = details.CacheRead
	item.CacheWrite = details.CacheWrite
	if details.Updated > 0 {
		item.UpdatedAt = time.UnixMilli(details.Updated)
	}
}

=======
>>>>>>> bb6579af04679718e00f1b5cb47321370f2e0353
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
<<<<<<< HEAD
		if !session.StartedAt.IsZero() {
			offset := proc.startedAt.Sub(session.StartedAt)
			if offset < 0 {
				offset = -offset
			}
			if offset > 2*time.Second {
				return fmt.Errorf("PID %d now belongs to a different process", session.PID)
			}
=======
		if session.Directory != "" && session.Directory != "unknown" && proc.cwd != "unknown" && proc.cwd != session.Directory {
			return fmt.Errorf("PID %d is now running in a different directory", session.PID)
>>>>>>> bb6579af04679718e00f1b5cb47321370f2e0353
		}

		return nil
	}

	return fmt.Errorf("PID %d is no longer an active OpenCode session", session.PID)
<<<<<<< HEAD
=======
}

type listedSession struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Updated   int64  `json:"updated"`
	Directory string `json:"directory"`
>>>>>>> bb6579af04679718e00f1b5cb47321370f2e0353
}

type listedSession struct {
	MatchedID    string `json:"matchedId"`
	MatchedTitle string `json:"matchedTitle"`
	ID           string `json:"id"`
	Title        string `json:"title"`
	Updated      int64  `json:"updated"`
	Directory    string `json:"directory"`
	Input        int64  `json:"inputTokens"`
	Output       int64  `json:"outputTokens"`
	Reasoning    int64  `json:"reasoningTokens"`
	CacheRead    int64  `json:"cacheReadTokens"`
	CacheWrite   int64  `json:"cacheWriteTokens"`
}

func (p *OpenCodeProvider) fetchSessionsByTitle(ctx context.Context, titles []string) (map[string]listedSession, error) {
	unique := make(map[string]struct{}, len(titles))
	quoted := make([]string, 0, len(titles))
	for _, title := range titles {
		title = strings.TrimSpace(title)
		if title == "" || len(title) > 500 {
			continue
		}
		if _, ok := unique[title]; ok {
			continue
		}
		unique[title] = struct{}{}
		quoted = append(quoted, "'"+strings.ReplaceAll(title, "'", "''")+"'")
	}
	if len(quoted) == 0 {
		return map[string]listedSession{}, nil
	}

	query := `SELECT title AS matchedTitle, id, title, directory, time_updated AS updated,
		tokens_input AS inputTokens, tokens_output AS outputTokens,
		tokens_reasoning AS reasoningTokens, tokens_cache_read AS cacheReadTokens,
		tokens_cache_write AS cacheWriteTokens
		FROM session
		WHERE parent_id IS NULL AND title IN (` + strings.Join(quoted, ",") + `)
		ORDER BY time_updated DESC`

	cmd := exec.CommandContext(ctx, p.binary, "db", query, "--format", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to match terminal session titles: %w", err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return map[string]listedSession{}, nil
	}

	var listed []listedSession
	if err := json.Unmarshal([]byte(trimmed), &listed); err != nil {
		return nil, fmt.Errorf("failed to decode terminal session matches: %w", err)
	}

	byTitle := make(map[string]listedSession, len(listed))
	for _, entry := range listed {
		if entry.MatchedTitle == "" {
			continue
		}
		if _, exists := byTitle[entry.MatchedTitle]; !exists {
			byTitle[entry.MatchedTitle] = entry
		}
	}
	return byTitle, nil
}

func (p *OpenCodeProvider) fetchSessionsByID(ctx context.Context, ids []string) (map[string]listedSession, error) {
	unique := make(map[string]struct{}, len(ids))
	quoted := make([]string, 0, len(ids))
	for _, id := range ids {
		if !isSessionID(id) {
			continue
		}
		if _, ok := unique[id]; ok {
			continue
		}
		unique[id] = struct{}{}
		quoted = append(quoted, "'"+id+"'")
	}
	if len(quoted) == 0 {
		return map[string]listedSession{}, nil
	}

	query := `SELECT s.id AS matchedId,
		COALESCE(parent.id, s.id) AS id,
		COALESCE(parent.title, s.title) AS title,
		COALESCE(parent.directory, s.directory) AS directory,
		COALESCE(parent.time_updated, s.time_updated) AS updated,
		COALESCE(parent.tokens_input, s.tokens_input) AS inputTokens,
		COALESCE(parent.tokens_output, s.tokens_output) AS outputTokens,
		COALESCE(parent.tokens_reasoning, s.tokens_reasoning) AS reasoningTokens,
		COALESCE(parent.tokens_cache_read, s.tokens_cache_read) AS cacheReadTokens,
		COALESCE(parent.tokens_cache_write, s.tokens_cache_write) AS cacheWriteTokens
		FROM session s
		LEFT JOIN session parent ON parent.id = s.parent_id
		WHERE s.id IN (` + strings.Join(quoted, ",") + `)`

	cmd := exec.CommandContext(ctx, p.binary, "db", query, "--format", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read session usage from opencode: %w", err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return map[string]listedSession{}, nil
	}

	var listed []listedSession
	if err := json.Unmarshal([]byte(trimmed), &listed); err != nil {
		return nil, fmt.Errorf("failed to decode opencode session usage: %w", err)
	}

	byMatch := make(map[string]listedSession, len(listed))
	for _, entry := range listed {
		if entry.MatchedID == "" {
			continue
		}
		byMatch[entry.MatchedID] = entry
	}

	return byMatch, nil
}

type processRecord struct {
	pid       int
	startedAt time.Time
	command   string
	memory    int64
	cpu       float64
}

func (p *OpenCodeProvider) fetchOpenCodeProcesses(ctx context.Context) ([]processRecord, error) {
	cmd := exec.CommandContext(ctx, "ps", "-axo", "pid,etime,%cpu,rss,command")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect processes: %w", err)
	}

	now := time.Now()
	lines := strings.Split(string(out), "\n")
	records := make([]processRecord, 0, 8)
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
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

		cpu, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			continue
		}

		rssKB, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			continue
		}

		command := strings.Join(fields[4:], " ")
		if !looksLikeOpenCodeSession(command) {
			continue
		}

		records = append(records, processRecord{
			pid:       pid,
			startedAt: now.Add(-time.Duration(elapsed) * time.Second),
			command:   command,
			memory:    rssKB * 1024,
			cpu:       cpu,
		})
	}

	return records, nil
}

type processSessionMatch struct {
	sessionID string
	title     string
	directory string
	runFound  bool
}

type runRecord struct {
	startedAt time.Time
	sessionID string
	directory string
}

func (p *OpenCodeProvider) matchActiveSessions(procs []processRecord) map[int]processSessionMatch {
	matches := make(map[int]processSessionMatch, len(procs))
	for _, proc := range procs {
		if id := sessionIDFromCommand(proc.command); id != "" {
			matches[proc.pid] = processSessionMatch{sessionID: id, runFound: true}
		}
	}

	logFile, err := os.Open(openCodeLogPath())
	if err == nil {
		defer logFile.Close()

		logMatches, _ := matchSessionsFromLog(logFile, procs)
		for pid, match := range logMatches {
			if matches[pid].sessionID == "" {
				matches[pid] = match
			}
		}
	}

	for _, proc := range procs {
		match := matches[proc.pid]
		if match.sessionID == "" {
			match.title = strings.TrimSpace(p.terminalTitles[proc.pid])
			matches[proc.pid] = match
		}
	}
	return matches
}

func matchSessionsFromLog(r io.Reader, procs []processRecord) (map[int]processSessionMatch, error) {
	runs := make(map[string]*runRecord)
	order := make([]string, 0, len(procs))
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		runID := logValue(line, "run")
		if runID == "" {
			continue
		}

		if strings.Contains(line, `message="creating instance"`) {
			stamp, err := time.Parse(time.RFC3339Nano, logValue(line, "timestamp"))
			if err == nil {
				runs[runID] = &runRecord{startedAt: stamp, directory: logValue(line, "directory")}
				order = append(order, runID)
			}
		}

		if sessionID := logValue(line, "session.id"); isSessionID(sessionID) {
			if run := runs[runID]; run != nil {
				run.sessionID = sessionID
			}
		}
	}

	type candidate struct {
		pid    int
		runID  string
		offset time.Duration
	}
	candidates := make([]candidate, 0, len(procs))
	for _, proc := range procs {
		for _, runID := range order {
			run := runs[runID]
			offset := proc.startedAt.Sub(run.startedAt)
			if offset < 0 {
				offset = -offset
			}
			if offset <= 10*time.Second {
				candidates = append(candidates, candidate{pid: proc.pid, runID: runID, offset: offset})
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].offset < candidates[j].offset })

	matches := make(map[int]processSessionMatch, len(procs))
	claimedRuns := make(map[string]struct{}, len(procs))
	for _, candidate := range candidates {
		if _, ok := matches[candidate.pid]; ok {
			continue
		}
		if _, ok := claimedRuns[candidate.runID]; ok {
			continue
		}
		run := runs[candidate.runID]
		matches[candidate.pid] = processSessionMatch{sessionID: run.sessionID, directory: run.directory, runFound: true}
		claimedRuns[candidate.runID] = struct{}{}
	}

	return matches, scanner.Err()
}

func openCodeLogPath() string {
	dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "opencode", "log", "opencode.log")
}

func logValue(line, key string) string {
	marker := key + "="
	start := strings.Index(line, marker)
	if start < 0 {
		return ""
	}
	value := line[start+len(marker):]
	if strings.HasPrefix(value, `"`) {
		value = value[1:]
		for i := 0; i < len(value); i++ {
			if value[i] == '"' && (i == 0 || value[i-1] != '\\') {
				return value[:i]
			}
		}
		return value
	}
	if end := strings.IndexByte(value, ' '); end >= 0 {
		value = value[:end]
	}
	return value
}

func sessionIDFromCommand(command string) string {
	fields := strings.Fields(command)
	for i, field := range fields {
		if (field == "-s" || field == "--session") && i+1 < len(fields) && isSessionID(fields[i+1]) {
			return fields[i+1]
		}
		if strings.HasPrefix(field, "--session=") {
			id := strings.TrimPrefix(field, "--session=")
			if isSessionID(id) {
				return id
			}
		}
	}
	return ""
}

func isSessionID(value string) bool {
	if !strings.HasPrefix(value, "ses_") || len(value) <= len("ses_") {
		return false
	}
	for _, r := range value[len("ses_"):] {
		if r < '0' || (r > '9' && r < 'A') || (r > 'Z' && r < 'a') || r > 'z' {
			return false
		}
	}
	return true
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
		"plugin":     {},
		"pr":         {},
		"providers":  {},
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
