package model

import (
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// configMu protects all YAML config file reads/writes
var configMu sync.RWMutex

// readYAML reads and unmarshals a YAML file into v.
// Uses RLock for thread-safe reads.
func readYAML(path string, v interface{}) error {
	configMu.RLock()
	defer configMu.RUnlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}

// writeYAML writes v to a YAML file atomically (temp file + rename).
// Uses Lock for thread-safe writes.
func writeYAML(path string, v interface{}) error {
	configMu.Lock()
	defer configMu.Unlock()

	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
