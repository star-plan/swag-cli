package tui

import (
	"context"
	"fmt"
	"os"
	"swag-cli/internal/config"
	"swag-cli/internal/docker"
	"swag-cli/internal/nginx"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

// Run 启动交互式向导
func Run(swagDir string, swagContainerName string, network string, version string) {
	if version == "" {
		version = "dev"
	}
	color.Cyan("swag-cli version: %s", version)
	fmt.Println()
	for {
		action := ""
		prompt := &survey.Select{
			Message: "请选择操作:",
			Options: []string{"添加新站点 (Add)", "查看站点列表 (List)", "退出 (Exit)"},
		}
		survey.AskOne(prompt, &action)

		switch action {
		case "添加新站点 (Add)":
			runAddFlow(swagDir, swagContainerName, network)
		case "查看站点列表 (List)":
			// TODO: 调用 list 逻辑
			color.Yellow("列表功能在交互模式下暂未完全集成，请使用 'swag-cli list' 命令")
		case "退出 (Exit)":
			os.Exit(0)
		}
		fmt.Println()
	}
}

func runAddFlow(swagDir string, swagContainerName string, network string) {
	// 1. 获取容器列表
	cli, err := docker.NewClient()
	if err != nil {
		color.Red("Docker 连接失败: %v", err)
		return
	}

	containers, err := cli.ListContainersByNetwork(context.Background(), network)
	if err != nil {
		color.Red("无法获取容器列表 (请确保容器已加入 '%s' 网络): %v", network, err)
		return
	}

	// 准备选项
	var options []string
	containerMap := make(map[string]docker.ContainerInfo)
	for _, c := range containers {
		label := fmt.Sprintf("%s (%s)", c.Name, c.IP)
		options = append(options, label)
		containerMap[label] = c
	}

	// 2. 选择容器
	selectedLabel := ""
	prompt := &survey.Select{
		Message: "选择目标容器:",
		Options: options,
	}
	if err := survey.AskOne(prompt, &selectedLabel); err != nil {
		return
	}

	selectedContainer := containerMap[selectedLabel]

	// 3. 收集配置信息
	var answers struct {
		Subdomain string
		Port      int
		Protocol  string
	}

	qs := []*survey.Question{
		{
			Name: "Subdomain",
			Prompt: &survey.Input{
				Message: "请输入子域名:",
				Default: selectedContainer.Name,
			},
			Validate: survey.Required,
		},
		{
			Name: "Port",
			Prompt: &survey.Input{
				Message: "容器端口:",
				Default: "80",
			},
			// Survey input for int is tricky, usually parse string.
			// Let's stick to string parsing or use a custom validator if needed.
			// survey/v2 handles basic types but Input returns string.
			// We will handle conversion later or use a struct tag with a transform?
			// survey decodes into struct fields.
		},
		{
			Name: "Protocol",
			Prompt: &survey.Select{
				Message: "协议:",
				Options: []string{"http", "https"},
				Default: "http",
			},
		},
	}

	// Create a temporary struct for survey to decode into
	// Since Port in Input is string, we need to handle it.
	// Actually survey can decode into int if the input string is a valid number.
	err = survey.Ask(qs, &answers)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 4. 生成配置
	cfg := config.Config{SwagDir: swagDir}
	gen := nginx.NewGenerator(cfg.ProxyConfsDir())
	data := nginx.ConfigData{
		Subdomain:     answers.Subdomain,
		ContainerName: selectedContainer.Name,
		ContainerPort: answers.Port,
		Protocol:      answers.Protocol,
	}

	path, err := gen.GenerateConfig(data)
	if err != nil {
		color.Red("生成失败: %v", err)
		return
	}

	color.Green("配置已生成: %s", path)

	// 5. Reload Nginx
	color.Yellow("正在重载 SWAG (%s) Nginx...", swagContainerName)
	if err := cli.ReloadNginx(context.Background(), swagContainerName); err != nil {
		color.Red("Nginx 重载失败: %v", err)
	} else {
		color.Green("Nginx 重载成功！站点应已生效。")
	}
}
