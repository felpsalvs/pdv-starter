// Package config resolves the on-disk locations the PDV binary uses: the
// SQLite database, the backups directory, the printer config file and the
// HTTP port. All of them are relative to the executable's own directory by
// default (so the .exe is self-contained wherever it's copied), but every
// path can be overridden with an environment variable — the same variables
// the Node backend already used (PDV_DB_PATH, PDV_BACKUP_DIR), so existing
// test setups keep working unchanged. PDV_HOME overrides the base directory
// itself, which is what `go run` uses in development since there's no
// meaningful executable path to anchor on.
package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	BaseDir       string
	DBPath        string
	BackupDir     string
	PrinterConfig string
	Port          string
}

// Load resolves the configuration from the environment and the running
// executable's location.
func Load() (Config, error) {
	base, err := baseDir()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		BaseDir:       base,
		DBPath:        envOr("PDV_DB_PATH", filepath.Join(base, "data", "pdv.db")),
		BackupDir:     envOr("PDV_BACKUP_DIR", filepath.Join(base, "backups")),
		PrinterConfig: filepath.Join(base, "printer.config.json"),
		Port:          envOr("PORT", "3000"),
	}
	return cfg, nil
}

func baseDir() (string, error) {
	if home := os.Getenv("PDV_HOME"); home != "" {
		return filepath.Abs(home)
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
