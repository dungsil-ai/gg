package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Config는 host별 provider 선택만 담는다. token 저장 금지.
type Config struct {
	Hosts map[string]string `json:"hosts"`
}

// ConfigDir: $GG_HOME → $XDG_CONFIG_HOME/gg → ~/.gg
func ConfigDir() string {
	if d := os.Getenv("GG_HOME"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "gg")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gg")
}

func ConfigPath() string { return filepath.Join(ConfigDir(), "config.json") }

func LoadConfig() (Config, error) {
	cfg := Config{Hosts: map[string]string{}}
	data, err := os.ReadFile(ConfigPath())
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("broken config %s: %v", ConfigPath(), err)
	}
	if cfg.Hosts == nil {
		cfg.Hosts = map[string]string{}
	}
	return cfg, nil
}

var (
	renameFile = os.Rename
	removeFile = os.Remove
	goos       = runtime.GOOS
)

func replaceConfigFile(tmpPath, dstPath string) error {
	if err := renameFile(tmpPath, dstPath); err != nil {
		if goos != "windows" {
			return err
		}
		if rmErr := removeFile(dstPath); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			return rmErr
		}
		return renameFile(tmpPath, dstPath)
	}
	return nil
}

// SaveProvider는 temp 파일 + rename으로 원자적으로 저장한다.
func SaveProvider(host string, p Provider) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err // 손상 파일은 절대 덮어쓰지 않는다
	}
	cfg.Hosts[host] = string(p)
	if err := os.MkdirAll(ConfigDir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(ConfigDir(), "config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return replaceConfigFile(tmp.Name(), ConfigPath())
}
