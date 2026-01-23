package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Config struct {
	SwagDir       string `json:"swagDir"`       // Base SWAG directory path
	SwagContainer string `json:"swagContainer"` // SWAG container name
	Network       string `json:"network"`       // Docker network name
}

func Default() Config {
	return Config{
		SwagDir:       "~/apps-docker/swag",
		SwagContainer: "swag",
		Network:       "swag",
	}
}

// ProxyConfsDir returns the path to nginx proxy-confs directory
func (c Config) ProxyConfsDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "nginx", "proxy-confs")
}

// SiteConfsDir returns the path to nginx site-confs directory
func (c Config) SiteConfsDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "nginx", "site-confs")
}

// DefaultSiteConfPath returns the path to nginx default site configuration file.
// It prefers "site-confs/default" and falls back to "site-conf/default" for compatibility.
func (c Config) DefaultSiteConfPath() (string, error) {
	p1 := filepath.Join(expandPath(c.SwagDir), "config", "nginx", "site-confs", "default")
	if _, err := os.Stat(p1); err == nil {
		return p1, nil
	}

	p2 := filepath.Join(expandPath(c.SwagDir), "config", "nginx", "site-conf", "default")
	if _, err := os.Stat(p2); err == nil {
		return p2, nil
	}

	if _, err := os.Stat(filepath.Join(expandPath(c.SwagDir), "config", "nginx")); err != nil {
		return "", fmt.Errorf("nginx config directory not found: %w", err)
	}

	return p1, fmt.Errorf("default site config not found (checked: %s, %s)", p1, p2)
}

// NginxConfigDir returns the path to nginx config directory
func (c Config) NginxConfigDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "nginx")
}

// LogDir returns the path to nginx log directory
func (c Config) LogDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "log", "nginx")
}

// SSLCertDir returns the path to SSL certificates directory
func (c Config) SSLCertDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "etc", "letsencrypt")
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	} else if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	return path
}

func Path() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户配置目录: %w", err)
	}
	return filepath.Join(baseDir, "swag-cli", "config.json"), nil
}

func Load() (Config, error) {
	cfg := Default()

	p, err := Path()
	if err != nil {
		return cfg, err
	}

	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("读取配置文件失败 (%s): %w", p, err)
	}

	if len(strings.TrimSpace(string(b))) == 0 {
		return cfg, nil
	}

	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("解析配置文件失败 (%s): %w", p, err)
	}

	cfg = normalize(cfg)
	return cfg, nil
}

func Save(cfg Config) error {
	p, err := Path()
	if err != nil {
		return err
	}

	cfg = normalize(cfg)

	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建配置目录失败 (%s): %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写入临时配置文件失败 (%s): %w", tmp, err)
	}

	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("保存配置失败 (%s): %w", p, err)
	}

	return nil
}

func Keys() []string {
	keys := []string{
		"swag-dir",
		"swag-container",
		"network",
	}
	sort.Strings(keys)
	return keys
}

func Get(cfg Config, key string) (string, bool) {
	switch normalizeKey(key) {
	case "swag-dir":
		return cfg.SwagDir, true
	case "swag-container":
		return cfg.SwagContainer, true
	case "network":
		return cfg.Network, true
	default:
		return "", false
	}
}

func Set(cfg *Config, key, value string) error {
	if cfg == nil {
		return errors.New("配置对象为空")
	}

	switch normalizeKey(key) {
	case "swag-dir":
		cfg.SwagDir = strings.TrimSpace(value)
		return nil
	case "swag-container":
		cfg.SwagContainer = strings.TrimSpace(value)
		return nil
	case "network":
		cfg.Network = strings.TrimSpace(value)
		return nil
	default:
		return fmt.Errorf("未知配置项: %s", key)
	}
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func normalize(cfg Config) Config {
	cfg.SwagDir = strings.TrimSpace(cfg.SwagDir)
	cfg.SwagContainer = strings.TrimSpace(cfg.SwagContainer)
	cfg.Network = strings.TrimSpace(cfg.Network)

	if cfg.SwagDir == "" {
		cfg.SwagDir = Default().SwagDir
	}
	if cfg.SwagContainer == "" {
		cfg.SwagContainer = Default().SwagContainer
	}
	if cfg.Network == "" {
		cfg.Network = Default().Network
	}

	return cfg
}
