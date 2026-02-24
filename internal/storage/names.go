package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

var (
	mu       sync.Mutex
	filePath string
)

func init() {
	home, _ := os.UserHomeDir()
	filePath = filepath.Join(home, ".claude", "session-names.json")
}

// LoadNames reads the session names file
func LoadNames() (map[string]string, error) {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}

	names := make(map[string]string)
	if err := json.Unmarshal(data, &names); err != nil {
		return make(map[string]string), nil
	}
	return names, nil
}

// SaveNames writes the session names file
func SaveNames(names map[string]string) error {
	mu.Lock()
	defer mu.Unlock()

	data, err := json.MarshalIndent(names, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// SetName sets a custom name for a session
func SetName(sessionID, name string) error {
	names, err := LoadNames()
	if err != nil {
		return err
	}
	names[sessionID] = name
	return SaveNames(names)
}

// DeleteName removes a custom name for a session
func DeleteName(sessionID string) error {
	names, err := LoadNames()
	if err != nil {
		return err
	}
	delete(names, sessionID)
	return SaveNames(names)
}
