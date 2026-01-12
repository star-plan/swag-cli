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
	ConfDir       string `json:"confDir"`
	SwagContainer string `json:"swagContainer"`
	Network       string `json:"network"`
}

func Default() Config {
	return Config{
		ConfDir:       "./proxy-confs",
		SwagContainer: "swag",
		Network:       "swag",
	}
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
		"conf-dir",
		"swag-container",
		"network",
	}
	sort.Strings(keys)
	return keys
}

func Get(cfg Config, key string) (string, bool) {
	switch normalizeKey(key) {
	case "conf-dir":
		return cfg.ConfDir, true
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
	case "conf-dir":
		cfg.ConfDir = strings.TrimSpace(value)
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
	cfg.ConfDir = strings.TrimSpace(cfg.ConfDir)
	cfg.SwagContainer = strings.TrimSpace(cfg.SwagContainer)
	cfg.Network = strings.TrimSpace(cfg.Network)

	if cfg.ConfDir == "" {
		cfg.ConfDir = Default().ConfDir
	}
	if cfg.SwagContainer == "" {
		cfg.SwagContainer = Default().SwagContainer
	}
	if cfg.Network == "" {
		cfg.Network = Default().Network
	}

	return cfg
}
