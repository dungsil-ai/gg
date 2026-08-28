package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofrs/flock"
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

// NormalizeConfigHost는 hostname 또는 hostname:port를 port 없는 lowercase host로 바꾼다.
func NormalizeConfigHost(input string) (string, error) {
	raw := strings.TrimSpace(input)
	if raw == "" || raw != input || strings.ContainsAny(raw, `/\\?#@`) || strings.Contains(raw, "://") {
		return "", usageErr("host must be a hostname or hostname:port")
	}
	if net.ParseIP(raw) != nil {
		return strings.ToLower(raw), nil
	}
	u, err := url.Parse("//" + raw)
	if err != nil || u.Host != raw || u.Hostname() == "" || strings.HasSuffix(raw, ":") {
		return "", usageErr("host must be a hostname or hostname:port")
	}
	host := u.Hostname()
	if strings.HasSuffix(host, ".") || net.ParseIP(host) == nil && !validDNSHostname(host) {
		return "", usageErr("host must be a valid hostname or IP address")
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", usageErr("host port must be between 1 and 65535")
		}
	}
	return strings.ToLower(host), nil
}

func validDNSHostname(host string) bool {
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || !isHostnameLetterOrDigit(label[0]) || !isHostnameLetterOrDigit(label[len(label)-1]) {
			return false
		}
		for i := 1; i < len(label)-1; i++ {
			if !isHostnameLetterOrDigit(label[i]) && label[i] != '-' {
				return false
			}
		}
	}
	return true
}

func isHostnameLetterOrDigit(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

func ParseProvider(input string) (Provider, error) {
	p := Provider(input)
	if p != GH && p != GLab && p != Tea {
		return "", usageErr("provider must be gh, glab, or tea")
	}
	return p, nil
}

func LoadConfig() (Config, error) {
	cfg := Config{Hosts: map[string]string{}}
	data, err := os.ReadFile(ConfigPath())
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return cfg, fmt.Errorf("broken config %s: %v", ConfigPath(), err)
	}
	hostsJSON, ok := document["hosts"]
	if document == nil || !ok || string(hostsJSON) == "null" {
		return cfg, fmt.Errorf("broken config %s: hosts must be an object", ConfigPath())
	}
	if err := json.Unmarshal(hostsJSON, &cfg.Hosts); err != nil || cfg.Hosts == nil {
		if err == nil {
			err = errors.New("hosts must be an object")
		}
		return Config{Hosts: map[string]string{}}, fmt.Errorf("broken config %s: %v", ConfigPath(), err)
	}
	for host, provider := range cfg.Hosts {
		normalized, hostErr := NormalizeConfigHost(host)
		_, providerErr := ParseProvider(provider)
		_, fixed := defaultProviders[host]
		if hostErr != nil || normalized != host || providerErr != nil || fixed {
			return Config{Hosts: map[string]string{}}, fmt.Errorf("broken config %s: invalid host or provider entry", ConfigPath())
		}
	}
	return cfg, nil
}

var renameConfigFile = os.Rename

// SaveProvider는 temp 파일 + rename으로 원자적으로 저장한다.
func SaveProvider(host string, p Provider) error {
	return withConfigLock(func() error {
		cfg, err := LoadConfig()
		if err != nil {
			return err // 손상 파일은 절대 덮어쓰지 않는다
		}
		cfg.Hosts[host] = string(p)
		return saveConfig(cfg)
	})
}

func UnsetProvider(host string) error {
	return withConfigLock(func() error {
		cfg, err := LoadConfig()
		if err != nil {
			return err // 손상 파일은 절대 덮어쓰지 않는다
		}
		if _, ok := cfg.Hosts[host]; !ok {
			return nil
		}
		delete(cfg.Hosts, host)
		return saveConfig(cfg)
	})
}

func withConfigLock(action func() error) (err error) {
	if err := os.MkdirAll(ConfigDir(), 0o700); err != nil {
		return err
	}
	lock := flock.New(ConfigPath() + ".lock")
	if err := lock.Lock(); err != nil {
		return err
	}
	defer func() {
		if unlockErr := lock.Unlock(); err == nil {
			err = unlockErr
		}
	}()
	return action()
}

func saveConfig(cfg Config) error {
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
	return renameConfigFile(tmp.Name(), ConfigPath())
}
