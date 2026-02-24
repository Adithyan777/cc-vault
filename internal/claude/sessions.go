package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Session represents a parsed Claude Code session
type Session struct {
	ID                string
	ProjectPath       string // encoded project dir name
	FilePath          string // full path to .jsonl
	Date              time.Time
	ConversationCount int    // -1 = unknown, computed lazily for prune
	Title             string // first user message or custom name
	CustomName        string // from session-names.json
	IsPinned          bool   // is the lastSessionId for this project
	GitBranch         string
	Selected          bool // for bulk operations
}

// jsonlLine is used for quick parsing of the first few fields
type jsonlLine struct {
	Type      string          `json:"type"`
	Message   json.RawMessage `json:"message"`
	IsMeta    bool            `json:"isMeta"`
	Timestamp string          `json:"timestamp"`
	GitBranch string          `json:"gitBranch"`
}

type messageContent struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// LoadSessions loads all sessions for a given project encoded directory name
func LoadSessions(encodedProject string, lastSessionID string) ([]Session, error) {
	projectDir := filepath.Join(claudeDir(), "projects", encodedProject)
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") || strings.HasPrefix(entry.Name(), "agent-") {
			continue
		}

		sessionID := strings.TrimSuffix(entry.Name(), ".jsonl")
		filePath := filepath.Join(projectDir, entry.Name())

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Quick scan: only reads first ~30 lines for title/date/branch
		title, date, branch := scanSessionQuick(filePath)
		if date.IsZero() {
			date = info.ModTime()
		}

		if title == "" {
			title = sessionID[:8] + "..."
		}

		// Truncate title for display
		if len(title) > 60 {
			title = title[:57] + "..."
		}

		// Read custom-title from tail of file (Claude's native rename format)
		customName := readCustomTitle(filePath)

		session := Session{
			ID:                sessionID,
			ProjectPath:       encodedProject,
			FilePath:          filePath,
			Date:              date,
			ConversationCount: -1, // unknown until prune scans it
			Title:             title,
			CustomName:        customName,
			IsPinned:          sessionID == lastSessionID,
			GitBranch:         branch,
		}

		sessions = append(sessions, session)
	}

	// Sort by date descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Date.After(sessions[j].Date)
	})

	return sessions, nil
}

// scanSessionQuick reads only the first ~30 lines to extract title, date, branch.
// It does NOT count messages - that's done lazily by CountConversationMessages.
func scanSessionQuick(path string) (title string, date time.Time, branch string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	foundTitle := false
	linesRead := 0
	for scanner.Scan() {
		linesRead++
		// Stop once we have everything or after 30 lines
		if (foundTitle && !date.IsZero() && branch != "") || linesRead > 30 {
			break
		}

		var line jsonlLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}

		if date.IsZero() && line.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339Nano, line.Timestamp); err == nil {
				date = t
			} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", line.Timestamp); err == nil {
				date = t
			}
		}

		if branch == "" && line.GitBranch != "" {
			branch = line.GitBranch
		}

		if !foundTitle && line.Type == "user" && !line.IsMeta && line.Message != nil {
			var mc messageContent
			if err := json.Unmarshal(line.Message, &mc); err == nil && mc.Role == "user" {
				text := extractText(mc.Content)
				if text == "" {
					continue
				}
				// Check if this is a slash command (starts with <command- tags)
				if strings.HasPrefix(text, "<command-") {
					cmdTitle := parseSlashCommandTitle(text)
					// Skip non-informative commands like /clear
					if cmdTitle != "" && cmdTitle != "/clear" {
						title = cmdTitle
						foundTitle = true
					}
				} else if !strings.HasPrefix(text, "<") {
					title = text
					foundTitle = true
				}
			}
		}
	}
	return
}

// CountConversationMessages does a full scan to count real user/assistant messages.
// Called lazily (e.g. only when prune is triggered).
func CountConversationMessages(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	for scanner.Scan() {
		var line jsonlLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if (line.Type == "user" || line.Type == "assistant") && !line.IsMeta && line.Message != nil {
			var mc messageContent
			if err := json.Unmarshal(line.Message, &mc); err == nil {
				text := extractText(mc.Content)
				if text != "" && (!strings.HasPrefix(text, "<") || strings.HasPrefix(text, "<command-")) {
					count++
				}
			}
		}
	}
	return count
}

// extractText gets text content from a message content field (string or content blocks array)
func extractText(raw json.RawMessage) string {
	// Try as string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}

	// Try as content block array
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return strings.TrimSpace(b.Text)
			}
		}
	}

	return ""
}

// parseSlashCommandTitle extracts a readable title from slash command XML content.
// Input format: "<command-name>/feature-dev</command-name>\n<command-args>some args</command-args>..."
// Output: "/feature-dev: some args" or just "/feature-dev" if no args.
func parseSlashCommandTitle(text string) string {
	name := extractTagValue(text, "command-name")
	args := extractTagValue(text, "command-args")

	if name == "" {
		return ""
	}

	if args != "" {
		return name + ": " + args
	}
	return name
}

// extractTagValue extracts the text content between <tag>...</tag>.
func extractTagValue(text, tag string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	start := strings.Index(text, open)
	if start == -1 {
		return ""
	}
	start += len(open)
	end := strings.Index(text[start:], close)
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(text[start : start+end])
}

// DisplayName returns the custom name if set, otherwise the title
func (s *Session) DisplayName() string {
	if s.CustomName != "" {
		return s.CustomName
	}
	return s.Title
}

// readCustomTitle reads the last custom-title entry from a JSONL file.
// Reads only the last ~8KB of the file for speed.
func readCustomTitle(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return ""
	}

	// Read last 8KB (custom-title lines are small, this is plenty)
	readSize := int64(8192)
	size := info.Size()
	offset := size - readSize
	if offset < 0 {
		offset = 0
		readSize = size
	}

	buf := make([]byte, readSize)
	n, err := f.ReadAt(buf, offset)
	if err != nil && n == 0 {
		return ""
	}
	buf = buf[:n]

	// Scan lines in the chunk, keep the last custom-title found
	var lastTitle string
	for _, line := range strings.Split(string(buf), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Quick check before parsing JSON
		if !strings.Contains(line, "custom-title") {
			continue
		}
		var entry struct {
			Type        string `json:"type"`
			CustomTitle string `json:"customTitle"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err == nil && entry.Type == "custom-title" {
			lastTitle = entry.CustomTitle
		}
	}

	return lastTitle
}

// WriteCustomTitle appends a custom-title entry to a session JSONL file.
// This is Claude Code's native rename format.
func WriteCustomTitle(filePath string, sessionID string, title string) error {
	entry := struct {
		Type        string `json:"type"`
		CustomTitle string `json:"customTitle"`
		SessionID   string `json:"sessionId"`
	}{
		Type:        "custom-title",
		CustomTitle: title,
		SessionID:   sessionID,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append([]byte("\n"), append(data, '\n')...))
	return err
}
