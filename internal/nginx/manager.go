package nginx

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	Name          string     // 站点名称 (subdomain)
	Filename      string     // 完整文件名
	Status        SiteStatus // 状态
	ContainerName string     // 代理指向的容器名 (从配置中解析)
	ContainerPort string     // 代理指向的端口 (从配置中解析)
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
			config := SiteConfig{
				Name:     subdomain,
				Filename: name,
				Status:   StatusEnabled,
			}
			m.parseConfigDetails(&config)
			sites = append(sites, config)
		} else if strings.HasSuffix(name, ".subdomain.conf.disabled") {
			subdomain := strings.TrimSuffix(name, ".subdomain.conf.disabled")
			config := SiteConfig{
				Name:     subdomain,
				Filename: name,
				Status:   StatusDisabled,
			}
			m.parseConfigDetails(&config)
			sites = append(sites, config)
		}
	}
	return sites, nil
}

// parseConfigDetails 解析配置文件内容以提取容器信息
func (m *Manager) parseConfigDetails(config *SiteConfig) {
	fullPath := filepath.Join(m.BasePath, config.Filename)
	file, err := os.Open(fullPath)
	if err != nil {
		return
	}
	defer file.Close()

	// 简单的正则匹配 set $upstream_app <name>; 和 set $upstream_port <port>;
	// 或者 proxy_pass
	// SWAG 模板通常使用:
	// set $upstream_app <container_name>;
	// set $upstream_port <port>;

	reApp := regexp.MustCompile(`set\s+\$upstream_app\s+([^;]+);`)
	rePort := regexp.MustCompile(`set\s+\$upstream_port\s+([^;]+);`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := reApp.FindStringSubmatch(line); len(matches) > 1 {
			config.ContainerName = strings.TrimSpace(matches[1])
		}
		if matches := rePort.FindStringSubmatch(line); len(matches) > 1 {
			config.ContainerPort = strings.TrimSpace(matches[1])
		}
	}
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
