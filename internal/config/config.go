package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// DNSConfDir returns the path to DNS provider configuration directory.
func (c Config) DNSConfDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "dns-conf")
}

// KeysDir returns the path to SWAG keys directory.
func (c Config) KeysDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "keys")
}

// LetsEncryptDir returns the path to Let's Encrypt directory (same as SSLCertDir).
func (c Config) LetsEncryptDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "etc", "letsencrypt")
}

// Fail2banDir returns the path to fail2ban directory.
func (c Config) Fail2banDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "fail2ban")
}

// CustomContInitDir returns the path to custom container init scripts directory.
func (c Config) CustomContInitDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "custom-cont-init.d")
}

// CustomServicesDir returns the path to custom services scripts directory.
func (c Config) CustomServicesDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "custom-services.d")
}

// CrontabsDir returns the path to crontabs directory.
func (c Config) CrontabsDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "crontabs")
}

// PHPDir returns the path to php configuration directory.
func (c Config) PHPDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "php")
}

// WWWDir returns the path to static web directory.
func (c Config) WWWDir() string {
	return filepath.Join(expandPath(c.SwagDir), "config", "www")
}

// ComposePath returns the path to compose.yaml in SWAG base directory.
func (c Config) ComposePath() string {
	return filepath.Join(expandPath(c.SwagDir), "compose.yaml")
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
	p, err := Path()
	if err != nil {
		return Default(), err
	}
	return LoadFrom(p)
}

// LoadFrom 从指定路径读取配置。
//
// - 文件不存在：返回默认配置，不返回错误（与 Load 保持一致）；
// - 文件为空：返回默认配置，不返回错误；
// - 其他读取/解析失败：返回默认配置 + 错误。
func LoadFrom(path string) (Config, error) {
	cfg := Default()
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "" {
		return cfg, errors.New("配置路径为空")
	}

	b, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("读取配置文件失败 (%s): %w", cleanPath, err)
	}

	if len(strings.TrimSpace(string(b))) == 0 {
		return cfg, nil
	}

	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("解析配置文件失败 (%s): %w", cleanPath, err)
	}

	cfg = normalize(cfg)
	return cfg, nil
}

func Save(cfg Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	return SaveTo(p, cfg)
}

// SaveTo 将配置保存到指定路径。
//
// 写入策略：先写入临时文件，再通过 rename 原子替换目标文件，以减少中断导致的半写入风险。
func SaveTo(path string, cfg Config) error {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "" {
		return errors.New("配置路径为空")
	}

	cfg = normalize(cfg)

	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建配置目录失败 (%s): %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	tmp := cleanPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写入临时配置文件失败 (%s): %w", tmp, err)
	}

	if err := os.Rename(tmp, cleanPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("保存配置失败 (%s): %w", cleanPath, err)
	}

	return nil
}

// ExportTo 将指定配置导出到目标文件。
//
// 与 SaveTo 的区别：ExportTo 面向“迁移/备份”场景，允许导出到任意位置与任意文件名。
func ExportTo(path string, cfg Config, pretty bool) error {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "" {
		return errors.New("导出路径为空")
	}

	cfg = normalize(cfg)

	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建导出目录失败 (%s): %w", dir, err)
	}

	var data []byte
	var err error
	if pretty {
		data, err = json.MarshalIndent(cfg, "", "  ")
	} else {
		data, err = json.Marshal(cfg)
	}
	if err != nil {
		return fmt.Errorf("序列化导出配置失败: %w", err)
	}

	tmp := cleanPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写入临时导出文件失败 (%s): %w", tmp, err)
	}

	if err := os.Rename(tmp, cleanPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("导出失败 (%s): %w", cleanPath, err)
	}

	return nil
}

// ImportFrom 从目标文件读取配置并返回规范化后的 Config。
//
// 与 LoadFrom 的区别：ImportFrom 用于“导入”操作——
// - 文件必须存在；文件为空视为错误；
// - 读取成功后会 normalize，保证字段完整可用。
func ImportFrom(path string) (Config, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "" {
		return Default(), errors.New("导入路径为空")
	}

	f, err := os.Open(cleanPath)
	if err != nil {
		return Default(), fmt.Errorf("打开导入文件失败 (%s): %w", cleanPath, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return Default(), fmt.Errorf("读取导入文件失败 (%s): %w", cleanPath, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return Default(), fmt.Errorf("导入文件为空 (%s)", cleanPath)
	}

	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("解析导入文件失败 (%s): %w", cleanPath, err)
	}
	cfg = normalize(cfg)
	return cfg, nil
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
