package tig

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// GlobalConfig is what holds for every project of one person: who they are. A
// project's own config says the same things, and the project's come first.
type GlobalConfig struct {
	UserName string `json:"userName,omitempty"` // who is acting
	UserBy   string `json:"userBy,omitempty"`   // what they say they are: human, ai, tool, or their own word
}

// GlobalConfigEnv names the variable that moves the global config elsewhere.
const GlobalConfigEnv = "TIG_CONFIG_GLOBAL"

// GlobalConfigPath is where the global config lives: ~/.tigconfig, or the file
// named by $TIG_CONFIG_GLOBAL.
func GlobalConfigPath() (string, error) {
	if p := os.Getenv(GlobalConfigEnv); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("tig: no home folder for the global config: %w", err)
	}
	return filepath.Join(home, ".tigconfig"), nil
}

// ReadGlobalConfig reads the global config. None is there: an empty one.
func ReadGlobalConfig() (GlobalConfig, error) {
	var c GlobalConfig
	file, err := GlobalConfigPath()
	if err != nil {
		return c, err
	}
	raw, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	rec, err := parseRecord(raw, configHeader)
	if err != nil {
		return c, fmt.Errorf("%s: %w", file, err)
	}
	c.UserName, c.UserBy = rec.get(keyUserName), rec.get(keyUserBy)
	return c, nil
}

// WriteGlobalConfig writes the global config.
func WriteGlobalConfig(c GlobalConfig) error {
	if err := checkIdentity(c.UserName, c.UserBy); err != nil {
		return err
	}
	file, err := GlobalConfigPath()
	if err != nil {
		return err
	}
	rec := newRecord(configHeader)
	rec.set(keyUserName, c.UserName)
	rec.set(keyUserBy, c.UserBy)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return writeAtomic(file, rec.bytes(), 0o644)
}

// The config keys of who someone is.
const (
	keyUserName = "user.name"
	keyUserBy   = "user.by"
)

// checkIdentity refuses what a record could not read back the same.
func checkIdentity(values ...string) error {
	for _, v := range values {
		if strings.ContainsAny(v, "\r\n") {
			return errors.New("tig: a config value is one line")
		}
	}
	return nil
}
