package tui

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"cc-sessions/internal/claude"
)

// SearchResult holds a matching session with match count
type SearchResult struct {
	SessionIndex int
	MatchCount   int
}

// SearchSessions searches through all session JSONL files in a project for the given query
func SearchSessions(sessions []claude.Session, query string) []SearchResult {
	if query == "" {
		return nil
	}

	query = strings.ToLower(query)
	var results []SearchResult

	for i, session := range sessions {
		count := searchInFile(session.FilePath, query)
		if count > 0 {
			results = append(results, SearchResult{
				SessionIndex: i,
				MatchCount:   count,
			})
		}
	}

	return results
}

// searchInFile counts occurrences of query in a JSONL file's message content
func searchInFile(filePath string, query string) int {
	f, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		// Quick check before parsing JSON
		if !strings.Contains(strings.ToLower(line), query) {
			continue
		}

		var entry struct {
			Type    string `json:"type"`
			IsMeta  bool   `json:"isMeta"`
			Message *struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}

		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry.Type != "user" && entry.Type != "assistant" {
			continue
		}
		if entry.IsMeta || entry.Message == nil {
			continue
		}

		text := extractSearchText(entry.Message.Content)
		if strings.Contains(strings.ToLower(text), query) {
			count++
		}
	}

	return count
}

func extractSearchText(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	var mc struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &mc); err == nil && mc.Content != nil {
		raw = mc.Content
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var texts []string
		for _, b := range blocks {
			if b.Type == "text" {
				texts = append(texts, b.Text)
			}
		}
		return strings.Join(texts, " ")
	}

	return string(raw)
}

// DeleteSessionFiles removes all files associated with a session
func DeleteSessionFiles(session *claude.Session) error {
	home, _ := os.UserHomeDir()
	claudeDir := filepath.Join(home, ".claude")

	// 1. Remove the main JSONL file
	os.Remove(session.FilePath)

	// 2. Remove agent files in the same project dir
	projectDir := filepath.Dir(session.FilePath)
	entries, _ := os.ReadDir(projectDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "agent-") && strings.Contains(e.Name(), session.ID) {
			os.Remove(filepath.Join(projectDir, e.Name()))
		}
	}

	// Also remove the session subdirectory if it exists (some sessions have dirs)
	os.RemoveAll(filepath.Join(projectDir, session.ID))

	// 3. Remove debug file
	os.Remove(filepath.Join(claudeDir, "debug", session.ID+".txt"))

	// 4. Remove file-history directory
	os.RemoveAll(filepath.Join(claudeDir, "file-history", session.ID))

	// 5. Remove session-env directory
	os.RemoveAll(filepath.Join(claudeDir, "session-env", session.ID))

	// 6. Remove todo files
	todosDir := filepath.Join(claudeDir, "todos")
	todoEntries, _ := os.ReadDir(todosDir)
	for _, e := range todoEntries {
		if strings.HasPrefix(e.Name(), session.ID) {
			os.Remove(filepath.Join(todosDir, e.Name()))
		}
	}

	return nil
}
