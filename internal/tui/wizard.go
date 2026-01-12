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
	// 1. 加载配置以获取 swag 容器名称
	cfg, err := config.Load()
	if err != nil {
		color.Red("加载配置失败: %v", err)
		return
	}

	// 使用配置中的 swag 容器名称，如果参数传入的为空
	if swagContainerName == "" {
		swagContainerName = cfg.SwagContainer
	}

	// 2. 获取容器列表
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

	// 3. 准备选项，排除 swag 容器本身
	var options []string
	containerMap := make(map[string]docker.ContainerInfo)
	for _, c := range containers {
		// 排除 swag 容器本身
		if c.Name == swagContainerName {
			continue
		}
		label := fmt.Sprintf("%s (%s)", c.Name, c.IP)
		options = append(options, label)
		containerMap[label] = c
	}

	// 检查是否有可用的容器
	if len(options) == 0 {
		color.Yellow("网络 '%s' 中没有可添加的容器（已排除 swag 容器 '%s'）", network, swagContainerName)
		return
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

	// 5. Restart SWAG Container
	color.Yellow("正在重启 SWAG 容器 (%s)...", swagContainerName)
	if err := cli.RestartContainer(context.Background(), swagContainerName); err != nil {
		color.Red("SWAG 容器重启失败: %v", err)
	} else {
		color.Green("SWAG 容器重启成功！站点应已生效。")
	}
}
