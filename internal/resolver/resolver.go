package resolver

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// exeDir returns the directory containing the running executable.
func exeDir() string {
	ex, err := os.Executable()
	if err != nil {
		return "."
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(ex))
	if err != nil {
		return filepath.Dir(ex)
	}
	return dir
}

// Resolve finds an external command binary.
// It first looks in the bin/ directory next to the executable,
// then falls back to the bare name (relies on PATH lookup).
// On Windows it automatically appends .exe if needed.
func Resolve(name string) string {
	binName := name
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
		binName = name + ".exe"
	}
	candidate := filepath.Join(exeDir(), "bin", binName)
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate
	}
	return name
}

// ModelDir returns the directory for whisper model files.
// Priority: exe-relative models/ > MODELS_DIR env > ~/.mcp-vtt/models > legacy ~/.my-vtt/models.
func ModelDir() string {
	// 1. exe-relative models/
	relDir := filepath.Join(exeDir(), "models")
	if _, err := os.Stat(relDir); err == nil {
		return relDir
	}
	// 2. env override
	if d := os.Getenv("MODELS_DIR"); d != "" {
		return d
	}
	// 3. ~/.mcp-vtt/models
	home, _ := os.UserHomeDir()
	newPath := filepath.Join(home, ".mcp-vtt", "models")
	if _, err := os.Stat(newPath); err == nil {
		return newPath
	}
	// 4. legacy ~/.my-vtt/models
	legacyPath := filepath.Join(home, ".my-vtt", "models")
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath
	}
	return newPath
}
