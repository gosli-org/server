package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveProjectDir(path, base string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("project directory must not be empty")
	}
	var home string
	if usesHome(path) {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home directory: %w", err)
		}
	}
	path = projectPath(path, base, home)
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("project directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project directory is not a directory")
	}
	return path, nil
}

func usesHome(path string) bool {
	return path == "~" || strings.HasPrefix(path, "~/")
}

// projectPath transforms a path without consulting the process or filesystem.
func projectPath(path, base, home string) string {
	switch {
	case usesHome(path):
		return filepath.Join(home, strings.TrimPrefix(path[1:], "/"))
	case !filepath.IsAbs(path):
		return filepath.Join(base, path)
	default:
		return filepath.Clean(path)
	}
}
