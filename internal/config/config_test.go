package config

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestIsValidTier(t *testing.T) {
	tests := []struct {
		name  string
		tier  string
		valid bool
	}{
		{"center is valid", "center", true},
		{"hub is valid", "hub", true},
		{"node is valid", "node", true},
		{"unknown is invalid", "unknown", false},
		{"empty is invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidTier(tt.tier); got != tt.valid {
				t.Errorf("IsValidTier(%q) = %v; want %v", tt.tier, got, tt.valid)
			}
		})
	}
}

func TestGetCacheDir(t *testing.T) {
	// 1. Custom dir
	customDir := filepath.Join(t.TempDir(), "custom_ctmux")
	got, err := GetCacheDir(customDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != customDir {
		t.Errorf("GetCacheDir(%q) = %q; want %q", customDir, got, customDir)
	}
	if info, err := os.Stat(got); err != nil || !info.IsDir() {
		t.Errorf("directory %q was not created", got)
	}

	// 2. Environment variable
	envDir := filepath.Join(t.TempDir(), "env_ctmux")
	t.Setenv("CTMUX_CACHE_DIR", envDir)
	got, err = GetCacheDir("")
	if err != nil {
		t.Fatalf("unexpected error with env: %v", err)
	}
	if got != envDir {
		t.Errorf("GetCacheDir with env = %q; want %q", got, envDir)
	}

	// 3. Default (without env)
	t.Setenv("CTMUX_CACHE_DIR", "")
	defaultDir, err := GetCacheDir("")
	if err != nil {
		t.Fatalf("unexpected error with default: %v", err)
	}
	if defaultDir == "" {
		t.Errorf("expected non-empty default cache dir")
	}
}

func TestEnsureConfigFile(t *testing.T) {
	tempDir := t.TempDir()

	mockFS := fstest.MapFS{
		"center.conf": &fstest.MapFile{Data: []byte("set-option -g prefix f7\n")},
		"hub.conf":    &fstest.MapFile{Data: []byte("set-option -g prefix f8\n")},
		"node.conf":   &fstest.MapFile{Data: []byte("set-option -g prefix C-x\n")},
	}

	// 1. Invalid tier
	if _, _, err := EnsureConfigFile(mockFS, tempDir, "invalid"); err == nil {
		t.Errorf("expected error for invalid tier, got nil")
	}

	// 2. Fresh creation
	path, updated, err := EnsureConfigFile(mockFS, tempDir, "center")
	if err != nil {
		t.Fatalf("failed to ensure center.conf: %v", err)
	}
	if !updated {
		t.Errorf("expected updated=true on fresh write")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != "set-option -g prefix f7\n" {
		t.Errorf("written content = %q; want %q", string(content), "set-option -g prefix f7\n")
	}

	// 3. Second call with unchanged content
	path2, updated2, err := EnsureConfigFile(mockFS, tempDir, "center")
	if err != nil {
		t.Fatalf("failed on second ensure: %v", err)
	}
	if updated2 {
		t.Errorf("expected updated=false when content matches")
	}
	if path2 != path {
		t.Errorf("paths do not match: %q vs %q", path2, path)
	}

	// 4. Content change in embedded FS -> updates file
	newMockFS := fstest.MapFS{
		"center.conf": &fstest.MapFile{Data: []byte("set-option -g prefix f7 # updated\n")},
	}
	path3, updated3, err := EnsureConfigFile(newMockFS, tempDir, "center")
	if err != nil {
		t.Fatalf("failed on third ensure: %v", err)
	}
	if !updated3 {
		t.Errorf("expected updated=true when embedded content changes")
	}
	updatedContent, err := os.ReadFile(path3)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}
	if string(updatedContent) != "set-option -g prefix f7 # updated\n" {
		t.Errorf("content was not updated on change")
	}
}

func TestResetCache(t *testing.T) {
	tempDir := t.TempDir()

	confPath := filepath.Join(tempDir, "center.conf")
	sockPath := filepath.Join(tempDir, "center.sock")
	keepPath := filepath.Join(tempDir, "other.txt")

	if err := os.WriteFile(confPath, []byte("conf"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sockPath, []byte("sock"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keepPath, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	removed, err := ResetCache(tempDir)
	if err != nil {
		t.Fatalf("ResetCache failed: %v", err)
	}

	if len(removed) != 2 {
		t.Errorf("removed %d files; want 2", len(removed))
	}

	if _, err := os.Stat(confPath); !os.IsNotExist(err) {
		t.Errorf("conf file still exists")
	}
	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Errorf("sock file still exists")
	}
	if _, err := os.Stat(keepPath); os.IsNotExist(err) {
		t.Errorf("unrelated file was removed")
	}
}

func TestConfigHash(t *testing.T) {
	data := []byte("hello world")
	hash := ConfigHash(data)
	if len(hash) != 16 {
		t.Errorf("expected 16 hex chars (8 bytes), got %d: %s", len(hash), hash)
	}
}
