package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SiteStatus 表示站点状态
type SiteStatus string

const (
	StatusEnabled  SiteStatus = "Enabled"
	StatusDisabled SiteStatus = "Disabled"
)

// SiteConfig 表示一个站点配置
type SiteConfig struct {
	Name     string     // 站点名称 (subdomain)
	Filename string     // 完整文件名
	Status   SiteStatus // 状态
}

// Manager 管理 Nginx 配置文件
type Manager struct {
	BasePath string
}

// NewManager 创建一个新的 Manager
func NewManager(basePath string) *Manager {
	return &Manager{BasePath: basePath}
}

// ListSites 列出所有站点配置
func (m *Manager) ListSites() ([]SiteConfig, error) {
	entries, err := os.ReadDir(m.BasePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []SiteConfig{}, nil
		}
		return nil, err
	}

	var sites []SiteConfig
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// 匹配 *.subdomain.conf 或 *.subdomain.conf.disabled
		// SWAG format: <subdomain>.subdomain.conf
		if strings.HasSuffix(name, ".subdomain.conf") {
			subdomain := strings.TrimSuffix(name, ".subdomain.conf")
			sites = append(sites, SiteConfig{
				Name:     subdomain,
				Filename: name,
				Status:   StatusEnabled,
			})
		} else if strings.HasSuffix(name, ".subdomain.conf.disabled") {
			subdomain := strings.TrimSuffix(name, ".subdomain.conf.disabled")
			sites = append(sites, SiteConfig{
				Name:     subdomain,
				Filename: name,
				Status:   StatusDisabled,
			})
		}
	}
	return sites, nil
}

// ToggleSite 切换站点状态
// subdomain: 站点名称
// enable: true 启用, false 禁用. 如果为 nil (toggle), 则反转当前状态 (这里简化逻辑，toggle 命令通常是 toggle 动作)
// 但为了明确，我们先实现 toggle 动作，或者根据当前文件名判断。
func (m *Manager) ToggleSite(subdomain string) (SiteStatus, error) {
	sites, err := m.ListSites()
	if err != nil {
		return "", err
	}

	var target *SiteConfig
	for _, s := range sites {
		if s.Name == subdomain {
			target = &s
			break
		}
	}

	if target == nil {
		return "", fmt.Errorf("site not found: %s", subdomain)
	}

	oldPath := filepath.Join(m.BasePath, target.Filename)
	var newFilename string
	var newStatus SiteStatus

	if target.Status == StatusEnabled {
		// Disable it
		newFilename = target.Name + ".subdomain.conf.disabled"
		newStatus = StatusDisabled
	} else {
		// Enable it
		newFilename = target.Name + ".subdomain.conf"
		newStatus = StatusEnabled
	}

	newPath := filepath.Join(m.BasePath, newFilename)
	if err := os.Rename(oldPath, newPath); err != nil {
		return "", fmt.Errorf("failed to rename file: %w", err)
	}

	return newStatus, nil
}
