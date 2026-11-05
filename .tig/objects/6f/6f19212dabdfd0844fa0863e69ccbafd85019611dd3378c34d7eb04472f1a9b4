//go:build !windows

package main

import (
	"os"
	"path/filepath"
)

// Other systems keep PATH in files the person writes, and have no list of
// installed programs for this to join. So there is nothing to change here: the
// setup says which line to add.

func defaultRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local"), nil
}

func registerPlan(string) []string   { return nil }
func unregisterPlan(string) []string { return nil }

func register(root, bin, program string, size int64, version string) error { return nil }

func unregister(root, bin string) error { return nil }

func removeAfterExit(self, root string) error { return os.Remove(self) }
