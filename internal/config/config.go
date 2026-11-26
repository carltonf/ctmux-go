package config

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// SupportedTiers lists the known ctmux session tiers.
var SupportedTiers = []string{"center", "hub", "node"}

// IsValidTier checks whether the given name is a valid ctmux tier.
func IsValidTier(name string) bool {
	for _, tier := range SupportedTiers {
		if tier == name {
			return true
		}
	}
	return false
}

// GetCacheDir resolves the directory to store ctmux configuration and socket files.
// Precedence:
//  1. customDir (if non-empty)
//  2. Environment variable CTMUX_CACHE_DIR
//  3. User cache directory via os.UserCacheDir() + "/ctmux"
//  4. Fallback: ~/.cache/ctmux or ./.cache
func GetCacheDir(customDir string) (string, error) {
	if customDir != "" {
		if err := os.MkdirAll(customDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create cache directory %q: %w", customDir, err)
		}
		return customDir, nil
	}

	if envDir := os.Getenv("CTMUX_CACHE_DIR"); envDir != "" {
		if err := os.MkdirAll(envDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create cache directory from CTMUX_CACHE_DIR %q: %w", envDir, err)
		}
		return envDir, nil
	}

	userCache, err := os.UserCacheDir()
	if err == nil && userCache != "" {
		dir := filepath.Join(userCache, "ctmux")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create user cache directory %q: %w", dir, err)
		}
		return dir, nil
	}

	// Fallback to ~/.cache/ctmux
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dir := filepath.Join(home, ".cache", "ctmux")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create fallback cache directory %q: %w", dir, err)
		}
		return dir, nil
	}

	// Last-resort fallback to local .cache
	localDir := ".cache"
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create local cache directory %q: %w", localDir, err)
	}
	return localDir, nil
}

// GetSocketPath returns the expected unix socket path for a given tier.
func GetSocketPath(cacheDir, name string) string {
	return filepath.Join(cacheDir, fmt.Sprintf("%s.sock", name))
}

// GetConfigPath returns the cached configuration file path for a given tier.
func GetConfigPath(cacheDir, name string) string {
	return filepath.Join(cacheDir, fmt.Sprintf("%s.conf", name))
}

// ReadEmbeddedConfig reads the content of an embedded config for a tier.
func ReadEmbeddedConfig(configFS fs.FS, name string) ([]byte, error) {
	filename := fmt.Sprintf("%s.conf", name)
	content, err := fs.ReadFile(configFS, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded config %q: %w", filename, err)
	}
	return content, nil
}

// EnsureConfigFile ensures that the configuration file for the given tier exists in
// the cache directory and is synchronized with the embedded configuration.
// If the file exists but its content differs (e.g. after a ctmux binary update),
// it is automatically refreshed.
func EnsureConfigFile(configFS fs.FS, cacheDir, name string) (string, bool, error) {
	if !IsValidTier(name) {
		return "", false, fmt.Errorf("invalid tier %q (expected one of: center, hub, node)", name)
	}

	embeddedContent, err := ReadEmbeddedConfig(configFS, name)
	if err != nil {
		return "", false, err
	}

	targetPath := GetConfigPath(cacheDir, name)

	existingContent, err := os.ReadFile(targetPath)
	if err == nil {
		// Compare contents
		if bytes.Equal(existingContent, embeddedContent) {
			return targetPath, false, nil
		}
	} else if !os.IsNotExist(err) {
		return "", false, fmt.Errorf("error reading existing config %q: %w", targetPath, err)
	}

	// Write or update the file
	if err := os.WriteFile(targetPath, embeddedContent, 0644); err != nil {
		return "", false, fmt.Errorf("failed to write config to %q: %w", targetPath, err)
	}

	return targetPath, true, nil
}

// ResetCache removes configuration and socket files in the cache directory.
// Returns a list of removed files.
func ResetCache(cacheDir string) ([]string, error) {
	var removed []string

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return removed, nil
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory %q: %w", cacheDir, err)
	}

	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if ext == ".conf" || ext == ".sock" {
			target := filepath.Join(cacheDir, entry.Name())
			if err := os.Remove(target); err != nil {
				return removed, fmt.Errorf("failed to remove %q: %w", target, err)
			}
			removed = append(removed, target)
		}
	}

	return removed, nil
}

// ConfigHash computes a hex SHA-256 hash of data for verification.
func ConfigHash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:8])
}
