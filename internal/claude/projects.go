package claude

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Project represents a discovered Claude Code project
type Project struct {
	// EncodedName is the directory name under ~/.claude/projects/ (e.g. "-Users-adithyankrishnan-Desktop-lifie")
	EncodedName string
	// FullPath is the decoded absolute path (e.g. "/Users/adithyankrishnan/Desktop/lifie")
	FullPath string
	// DisplayPath is the shortened display path (e.g. "~/Desktop/lifie")
	DisplayPath string
	// LastModified is the most recent session file mtime (unix timestamp)
	LastModified int64
}

// DecodePath converts encoded dir name to absolute path
// "-Users-adithyankrishnan-Desktop-lifie" → "/Users/adithyankrishnan/Desktop/lifie"
func DecodePath(encoded string) string {
	// Replace leading dash with /
	if strings.HasPrefix(encoded, "-") {
		encoded = "/" + encoded[1:]
	}
	// Replace remaining dashes with /
	return strings.ReplaceAll(encoded, "-", "/")
}

// ShortenPath replaces home dir prefix with ~
func ShortenPath(fullPath string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(fullPath, home) {
		return "~" + fullPath[len(home):]
	}
	return fullPath
}

// DiscoverProjects scans ~/.claude/projects/ and returns all discovered projects
func DiscoverProjects() ([]Project, error) {
	projectsDir := filepath.Join(claudeDir(), "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	var projects []Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		fullPath := DecodePath(name)
		displayPath := ShortenPath(fullPath)

		// Find most recent session file for sorting
		var latestMod int64
		sessionDir := filepath.Join(projectsDir, name)
		sessionEntries, err := os.ReadDir(sessionDir)
		if err != nil {
			continue
		}
		for _, se := range sessionEntries {
			if !se.IsDir() && strings.HasSuffix(se.Name(), ".jsonl") && !strings.HasPrefix(se.Name(), "agent-") {
				info, err := se.Info()
				if err == nil && info.ModTime().Unix() > latestMod {
					latestMod = info.ModTime().Unix()
				}
			}
		}

		projects = append(projects, Project{
			EncodedName:  name,
			FullPath:     fullPath,
			DisplayPath:  displayPath,
			LastModified: latestMod,
		})
	}

	// Sort by most recently modified
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].LastModified > projects[j].LastModified
	})

	return projects, nil
}

// FindProjectIndex returns the index of the project matching the given CWD, or 0
func FindProjectIndex(projects []Project, cwd string) int {
	for i, p := range projects {
		if p.FullPath == cwd {
			return i
		}
	}
	return 0
}
