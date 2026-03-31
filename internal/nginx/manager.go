package nginx

import (
	"bufio"
	"fmt"
	"net"
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
	TypeHomepage  SiteType = "Homepage"
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
	UpstreamProto string     // 代理协议 (从配置中解析)
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

	if homepage := m.readHomepageSite(); homepage != nil {
		sites = append(sites, *homepage)
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
	reProto := regexp.MustCompile(`set\s+\$upstream_proto\s+([^;]+);`)
	reRoot := regexp.MustCompile(`^\s*root\s+([^;]+);`)

	var upstreamApp, upstreamPort, upstreamProto, rootPath string

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
		if matches := reProto.FindStringSubmatch(line); len(matches) > 1 {
			upstreamProto = strings.TrimSpace(matches[1])
		}
		if matches := reRoot.FindStringSubmatch(line); len(matches) > 1 {
			rootPath = strings.TrimSpace(matches[1])
		}
	}

	config.ContainerPort = upstreamPort
	config.UpstreamProto = upstreamProto

	// 判定 TargetType
	if upstreamApp != "" {
		config.TargetDest = upstreamApp
		if config.UpstreamProto == "" {
			config.UpstreamProto = "http"
		}
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

func (m *Manager) readHomepageSite() *SiteConfig {
	defaultPath, err := m.defaultSiteConfPath()
	if err != nil {
		return nil
	}

	content, err := os.ReadFile(defaultPath)
	if err != nil {
		return nil
	}

	site := SiteConfig{
		Name:     "(homepage)",
		Type:     TypeHomepage,
		Filename: filepath.Base(defaultPath),
		Status:   StatusEnabled,
	}
	if !parseHomepageDetails(string(content), &site) {
		return nil
	}
	return &site
}

func (m *Manager) defaultSiteConfPath() (string, error) {
	nginxDir := filepath.Dir(m.BasePath)
	candidates := []string{
		filepath.Join(nginxDir, "site-confs", "default"),
		filepath.Join(nginxDir, "site-conf", "default"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", os.ErrNotExist
}

func parseHomepageDetails(content string, site *SiteConfig) bool {
	if site == nil {
		return false
	}

	reApp := regexp.MustCompile(`set\s+\$upstream_app\s+([^;]+);`)
	rePort := regexp.MustCompile(`set\s+\$upstream_port\s+([^;]+);`)
	reProto := regexp.MustCompile(`set\s+\$upstream_proto\s+([^;]+);`)
	reServerName := regexp.MustCompile(`^\s*server_name\s+([^;]+);`)

	var serverName string
	for _, rawLine := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if matches := reApp.FindStringSubmatch(line); len(matches) > 1 {
			site.TargetDest = strings.TrimSpace(matches[1])
			site.ContainerName = site.TargetDest
		}
		if matches := rePort.FindStringSubmatch(line); len(matches) > 1 {
			site.ContainerPort = strings.TrimSpace(matches[1])
		}
		if matches := reProto.FindStringSubmatch(line); len(matches) > 1 {
			site.UpstreamProto = strings.TrimSpace(matches[1])
		}
		if matches := reServerName.FindStringSubmatch(line); len(matches) > 1 {
			serverName = strings.TrimSpace(matches[1])
		}
	}

	if site.TargetDest == "" {
		return false
	}
	if site.UpstreamProto == "" {
		site.UpstreamProto = "http"
	}
	if isLikelyIP(site.TargetDest) {
		site.TargetType = TargetIP
	} else {
		site.TargetType = TargetContainer
	}
	if serverName != "" && serverName != "_" {
		site.Name = serverName
	}

	return true
}

func isLikelyIP(s string) bool {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	return net.ParseIP(s) != nil
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
	if target.Type == TypeHomepage {
		return "", fmt.Errorf("homepage is managed via 'swag-cli homepage set/clear'")
	}

	oldPath := filepath.Join(m.BasePath, target.Filename)
	var newFilename string
	var newStatus SiteStatus
	suffix := siteConfigSuffix(target.Type)

	if target.Status == StatusEnabled {
		// Disable it
		newFilename = target.Name + suffix + ".disabled"
		newStatus = StatusDisabled
	} else {
		// Enable it
		newFilename = target.Name + suffix
		newStatus = StatusEnabled
	}

	newPath := filepath.Join(m.BasePath, newFilename)
	if err := os.Rename(oldPath, newPath); err != nil {
		return "", fmt.Errorf("failed to rename file: %w", err)
	}

	return newStatus, nil
}

func siteConfigSuffix(siteType SiteType) string {
	switch siteType {
	case TypeSubfolder:
		return ".subfolder.conf"
	default:
		return ".subdomain.conf"
	}
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
	if target.Type == TypeHomepage {
		return fmt.Errorf("homepage is managed via 'swag-cli homepage set/clear'")
	}

	filePath := filepath.Join(m.BasePath, target.Filename)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
