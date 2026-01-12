package nginx

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"swag-cli/templates"
	"text/template"
)

// ConfigData 用于渲染模板的数据
type ConfigData struct {
	Subdomain     string
	ContainerName string
	ContainerPort int
	Protocol      string // http or https
	ExtraConfig   string
}

// Generator 处理 Nginx 配置文件生成
type Generator struct {
	BasePath string // SWAG proxy-confs 目录路径
}

// NewGenerator 创建一个新的 Generator
func NewGenerator(basePath string) *Generator {
	return &Generator{BasePath: basePath}
}

// GenerateConfig 生成并写入配置文件
func (g *Generator) GenerateConfig(data ConfigData) (string, error) {
	// 确保目录存在
	if _, err := os.Stat(g.BasePath); os.IsNotExist(err) {
		return "", fmt.Errorf("config directory does not exist: %s", g.BasePath)
	}

	// 准备模板
	tmpl, err := template.New("proxy").Parse(templates.StandardProxyTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// 构建文件名: <subdomain>.subdomain.conf
	filename := fmt.Sprintf("%s.subdomain.conf", data.Subdomain)
	fullPath := filepath.Join(g.BasePath, filename)

	// 检查文件是否已存在
	if _, err := os.Stat(fullPath); err == nil {
		return "", fmt.Errorf("file already exists: %s", fullPath)
	}

	// 写入文件
	if err := os.WriteFile(fullPath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return fullPath, nil
}
