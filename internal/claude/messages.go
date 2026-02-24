package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Message represents a parsed user/assistant message for preview
type Message struct {
	Role    string // "user" or "assistant"
	Content string // extracted text content
}

// PreviewData holds the data for the preview panel
type PreviewData struct {
	Messages      []Message // all conversation messages
	TotalMessages int       // total user+assistant message count
	GitBranch     string
}

// LoadPreview reads a session JSONL and extracts first/last messages for preview
func LoadPreview(filePath string) (*PreviewData, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var allMessages []Message
	var branch string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 2*1024*1024), 2*1024*1024) // 2MB buffer

	for scanner.Scan() {
		var line jsonlLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}

		if branch == "" && line.GitBranch != "" {
			branch = line.GitBranch
		}

		// Skip non-message types
		if line.Type != "user" && line.Type != "assistant" {
			continue
		}

		// Skip meta messages
		if line.IsMeta {
			continue
		}

		if line.Message == nil {
			continue
		}

		var mc messageContent
		if err := json.Unmarshal(line.Message, &mc); err != nil {
			continue
		}

		text := extractText(mc.Content)
		if text == "" || strings.HasPrefix(text, "<") {
			continue
		}

		allMessages = append(allMessages, Message{
			Role:    mc.Role,
			Content: text,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	preview := &PreviewData{
		Messages:      allMessages,
		TotalMessages: len(allMessages),
		GitBranch:     branch,
	}

	return preview, nil
}

// ExportSession exports a session to markdown format
func ExportSession(session *Session, projectDisplay string) (string, error) {
	preview, err := LoadPreview(session.FilePath)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	name := session.DisplayName()
	sb.WriteString(fmt.Sprintf("# Session: %s\n", name))
	sb.WriteString(fmt.Sprintf("**Date:** %s  **Project:** %s", session.Date.Format("2006-01-02"), projectDisplay))
	if preview.GitBranch != "" {
		sb.WriteString(fmt.Sprintf("  **Branch:** %s", preview.GitBranch))
	}
	sb.WriteString("\n\n---\n\n")

	// Load all messages for export
	f, err := os.Open(session.FilePath)
	if err != nil {
		return sb.String(), nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 2*1024*1024), 2*1024*1024)

	for scanner.Scan() {
		var line jsonlLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}

		if line.Type != "user" && line.Type != "assistant" {
			continue
		}
		if line.IsMeta {
			continue
		}
		if line.Message == nil {
			continue
		}

		var mc messageContent
		if err := json.Unmarshal(line.Message, &mc); err != nil {
			continue
		}

		text := extractText(mc.Content)
		if text == "" || strings.HasPrefix(text, "<") {
			continue
		}

		role := "User"
		if mc.Role == "assistant" {
			role = "Assistant"
		}

		sb.WriteString(fmt.Sprintf("**%s:** %s\n\n", role, text))
	}

	return sb.String(), nil
}
