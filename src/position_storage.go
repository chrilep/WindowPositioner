package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sys/windows/registry"
)

// PositionStorage manages the storage of window positions.
// It uses a JSON file to save and load positions, and can also interact with the Windows registry for startup settings.
type PositionStorage struct {
	//registryPath string
	storageFile string
	mu          sync.Mutex
}

// NewPositionStorage initializes a new PositionStorage instance.
// It creates the necessary directory for storing positions and initializes the storage file.
func NewPositionStorage() *PositionStorage {
	debug := true
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.Getenv("TEMP")
		if appData == "" {
			appData = "."
		}
	}
	dirPath := filepath.Join(appData, strPublisherName, strProductName)
	log(debug, "PositionStorage is using directory:", dirPath)
	_ = os.MkdirAll(dirPath, 0o755)

	return &PositionStorage{
		//registryPath: `Software\` + strPublisherName + `\` + strProductName,
		storageFile: filepath.Join(dirPath, "positions.json"),
	}
}

// SavePosition saves the position of a window identified by its identifier.
// The identifier is a unique string that combines the window's title, class name, executable,
// style, and extended style.
// It serializes the position to a JSON file.
func (ps *PositionStorage) SavePosition(identifier string, pos WindowPosition) error {
	positions, err := ps.loadAll()
	if err != nil {
		return fmt.Errorf("failed to load positions: %v", err)
	}
	positions[identifier] = pos
	return ps.saveAll(positions)
}

// LoadPosition retrieves the position of a window by its identifier.
// It deserializes the position from the JSON file.
func (ps *PositionStorage) LoadPosition(identifier string) (*WindowPosition, error) {
	positions, err := ps.loadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to load positions: %v", err)
	}
	pos, ok := positions[identifier]
	if !ok {
		return nil, fmt.Errorf("position not found for identifier '%s'", identifier)
	}
	return &pos, nil
}

// DeletePosition removes a window's position from storage by its identifier.
// It updates the JSON file to reflect the deletion.
func (ps *PositionStorage) DeletePosition(identifier string) error {
	positions, err := ps.loadAll()
	if err != nil {
		return fmt.Errorf("failed to load positions: %v", err)
	}
	delete(positions, identifier)
	return ps.saveAll(positions)
}

// GetAllPositions retrieves all saved window positions.
// It returns a map where the keys are identifiers and the values are WindowPosition structs.
func (ps *PositionStorage) GetAllPositions() map[string]WindowPosition {
	positions, err := ps.loadAll()
	if err != nil {
		return make(map[string]WindowPosition)
	}
	return positions
}

// loadAll reads all positions from the JSON file and returns them as a map.
// If the file does not exist, it returns an empty map.
func (ps *PositionStorage) loadAll() (map[string]WindowPosition, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	positions := make(map[string]WindowPosition)

	data, err := os.ReadFile(ps.storageFile)
	if err != nil {
		if os.IsNotExist(err) {
			return positions, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &positions); err != nil {
		return nil, err
	}
	return positions, nil
}

// saveAll writes all positions to the JSON file.
// It serializes the map of positions to JSON and saves it to the storage file.
func (ps *PositionStorage) saveAll(positions map[string]WindowPosition) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	data, err := json.MarshalIndent(positions, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := ps.storageFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpFile, ps.storageFile)
}

// EnableStartup adds the application to the Windows startup registry key.
// This allows the application to start automatically when the user logs in.
func EnableStartup() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	appName := strProductName
	// Fix: Use double quotes properly
	return key.SetStringValue(appName, `"`+exePath+`"`)
}

// DisableStartup removes the application from the Windows startup registry key.
// This prevents the application from starting automatically when the user logs in.
func DisableStartup() error {
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	appName := strProductName
	return key.DeleteValue(appName)
}

// IsStartupEnabled checks if the application is set to start with Windows.
func IsStartupEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.READ)
	if err != nil {
		return false
	}
	defer key.Close()

	appName := strProductName
	_, _, err = key.GetStringValue(appName)
	return err == nil
}
