package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func resolveAppDataPath(parts ...string) (string, error) {
	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("APPDATA")
	}
	if base == "" {
		userConfigDir, err := os.UserConfigDir()
		if err != nil || userConfigDir == "" {
			if err == nil {
				err = errors.New("user config directory is empty")
			}
			return "", fmt.Errorf("resolveAppDataPath: APPDATA not set and fallback lookup failed: %w", err)
		}
		base = userConfigDir
	}

	components := append([]string{base, "svrouter"}, parts...)
	return filepath.Join(components...), nil
}
