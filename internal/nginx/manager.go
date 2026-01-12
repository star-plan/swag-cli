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

// SiteType 表示站点类型
type SiteType string

const (
	TypeSubdomain SiteType = "Subdomain"
	TypeSubfolder SiteType = "Subfolder"
)

// TargetType 表示代理目标类型
type TargetType string

const (
	TargetContainer TargetType = "Container"
	TargetIP        TargetType = "IP"
	TargetStatic    TargetType = "Static"
	TargetOther     TargetType = "Other"
)

// SiteConfig 表示一个站点配置
type SiteConfig struct {
	Name          string     // 站点名称 (subdomain)
	Type          SiteType   // 站点类型
	Filename      string     // 完整文件名
	Status        SiteStatus // 状态
	TargetType    TargetType // 目标类型
	TargetDest    string     // 目标值 (容器名, IP, 路径等)
	ContainerName string     // (Legacy) 兼容旧代码，同 TargetDest (如果是容器)
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
		var siteType SiteType
		var status SiteStatus = StatusEnabled
		var configName string

		if strings.HasSuffix(name, ".subdomain.conf") {
			siteType = TypeSubdomain
			configName = strings.TrimSuffix(name, ".subdomain.conf")
		} else if strings.HasSuffix(name, ".subdomain.conf.disabled") {
			siteType = TypeSubdomain
			status = StatusDisabled
			configName = strings.TrimSuffix(name, ".subdomain.conf.disabled")
		} else if strings.HasSuffix(name, ".subfolder.conf") {
			siteType = TypeSubfolder
			configName = strings.TrimSuffix(name, ".subfolder.conf")
		} else if strings.HasSuffix(name, ".subfolder.conf.disabled") {
			siteType = TypeSubfolder
			status = StatusDisabled
			configName = strings.TrimSuffix(name, ".subfolder.conf.disabled")
		} else {
			continue
		}

		config := SiteConfig{
			Name:     configName,
			Type:     siteType,
			Filename: name,
			Status:   status,
		}
		m.parseConfigDetails(&config)
		sites = append(sites, config)
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

	// 简单的正则匹配
	reApp := regexp.MustCompile(`set\s+\$upstream_app\s+([^;]+);`)
	rePort := regexp.MustCompile(`set\s+\$upstream_port\s+([^;]+);`)
	reRoot := regexp.MustCompile(`^\s*root\s+([^;]+);`)

	var upstreamApp, upstreamPort, rootPath string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}

		if matches := reApp.FindStringSubmatch(line); len(matches) > 1 {
			upstreamApp = strings.TrimSpace(matches[1])
		}
		if matches := rePort.FindStringSubmatch(line); len(matches) > 1 {
			upstreamPort = strings.TrimSpace(matches[1])
		}
		if matches := reRoot.FindStringSubmatch(line); len(matches) > 1 {
			rootPath = strings.TrimSpace(matches[1])
		}
	}

	config.ContainerPort = upstreamPort

	// 判定 TargetType
	if upstreamApp != "" {
		config.TargetDest = upstreamApp
		// 简单启发式判断是否为IP (包含点且第一位是数字)
		if isLikelyIP(upstreamApp) {
			config.TargetType = TargetIP
		} else {
			config.TargetType = TargetContainer
			config.ContainerName = upstreamApp
		}
	} else if rootPath != "" {
		config.TargetType = TargetStatic
		config.TargetDest = rootPath
	} else {
		config.TargetType = TargetOther
		config.TargetDest = "Unknown"
	}
}

func isLikelyIP(s string) bool {
	if strings.Count(s, ".") == 3 {
		return true
	}
	// TODO: 更严谨的IP检测，这里暂时简单处理
	return false
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

// DeleteSite 删除站点配置
func (m *Manager) DeleteSite(subdomain string) error {
	sites, err := m.ListSites()
	if err != nil {
		return err
	}

	var target *SiteConfig
	for _, s := range sites {
		if s.Name == subdomain {
			target = &s
			break
		}
	}

	if target == nil {
		return fmt.Errorf("site not found: %s", subdomain)
	}

	filePath := filepath.Join(m.BasePath, target.Filename)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
