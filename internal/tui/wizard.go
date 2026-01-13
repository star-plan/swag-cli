package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
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
			runListFlow(swagDir, swagContainerName, network)
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
	restartSwagContainer(swagContainerName)
}

func runListFlow(swagDir string, swagContainerName string, network string) {
	for {
		cfg := config.Config{SwagDir: swagDir}
		manager := nginx.NewManager(cfg.ProxyConfsDir())
		sites, err := manager.ListSites()
		if err != nil {
			color.Red("读取配置文件失败: %v", err)
			return
		}

		if len(sites) == 0 {
			color.Yellow("未找到任何站点配置 (在 %s)", cfg.ProxyConfsDir())
			return
		}

		// 获取 Docker 容器信息以显示状态
		containerMap := make(map[string]docker.ContainerInfo)
		cli, err := docker.NewClient()
		dockerConnected := false
		if err == nil {
			dockerConnected = true
			containers, err := cli.ListContainersByNetwork(context.Background(), network)
			if err == nil {
				for _, c := range containers {
					containerMap[c.Name] = c
				}
			}
		}

		// 分组
		var containerSites, staticSites, otherSites, disabledSites []nginx.SiteConfig
		for _, site := range sites {
			if site.Status == nginx.StatusDisabled {
				disabledSites = append(disabledSites, site)
				continue
			}
			switch site.TargetType {
			case nginx.TargetContainer:
				containerSites = append(containerSites, site)
			case nginx.TargetStatic:
				staticSites = append(staticSites, site)
			default:
				otherSites = append(otherSites, site)
			}
		}

		var options []string
		siteMap := make(map[string]nginx.SiteConfig)

		// 辅助函数：生成标签并添加到选项
		addSites := func(groupName string, groupSites []nginx.SiteConfig) {
			if len(groupSites) == 0 {
				return
			}
			// 添加分组标题 (用特殊字符标记，处理选择时忽略)
			header := fmt.Sprintf("─── %s ───", groupName)
			options = append(options, header)

			for _, site := range groupSites {
				containerStatus := ""
				statusIcon := "🟢" // 默认绿色表示 Nginx 配置启用

				if site.Status == nginx.StatusDisabled {
					statusIcon = "🔴" // Disabled 显式红色
				}

				if dockerConnected && site.TargetType == nginx.TargetContainer {
					if _, ok := containerMap[site.ContainerName]; ok {
						containerStatus = "(在线)"
					} else {
						containerStatus = "(离线)"
						// 如果配置是 Enabled 但容器离线，使用黄色警告
						if site.Status == nginx.StatusEnabled {
							statusIcon = "🟡"
						}
					}
				} else if site.TargetType == nginx.TargetStatic {
					containerStatus = "(静态)"
				}

				dest := fmt.Sprintf("%s:%s", site.ContainerName, site.ContainerPort)
				if site.TargetType == nginx.TargetStatic {
					dest = site.TargetDest // Show root path for static sites
				}

				label := fmt.Sprintf("%s %-20s -> %-30s %s", statusIcon, site.Name, dest, containerStatus)
				options = append(options, label)
				siteMap[label] = site
			}
		}

		addSites("容器 (Containers)", containerSites)
		addSites("静态 (Static)", staticSites)
		addSites("其他 (Others)", otherSites)
		addSites("已禁用 (Disabled)", disabledSites)

		options = append(options, "返回主菜单 (Back)")

		selectedLabel := ""
		prompt := &survey.Select{
			Message:  "选择站点查看详情或操作:",
			Options:  options,
			PageSize: 20, // 增加每页显示数量以容纳分组标题
		}
		if err := survey.AskOne(prompt, &selectedLabel); err != nil {
			return
		}

		// 处理分组标题选择 (忽略并重试)
		if strings.HasPrefix(selectedLabel, "───") {
			continue
		}

		if selectedLabel == "返回主菜单 (Back)" {
			return
		}

		selectedSite := siteMap[selectedLabel]
		runSiteActionFlow(selectedSite, manager, swagContainerName)
	}
}

func runSiteActionFlow(site nginx.SiteConfig, manager *nginx.Manager, swagContainerName string) {
	// 显示详情
	fmt.Println()
	color.Cyan("站点详情:")
	fmt.Printf("  域名: %s\n", site.Name)
	fmt.Printf("  状态: %s\n", site.Status)
	fmt.Printf("  容器: %s\n", site.ContainerName)
	fmt.Printf("  端口: %s\n", site.ContainerPort)
	fmt.Printf("  文件: %s\n", site.Filename)
	fmt.Println()

	action := ""
	options := []string{"返回 (Back)"}
	if site.Status == nginx.StatusEnabled {
		options = append(options, "禁用站点 (Disable)")
	} else {
		options = append(options, "启用站点 (Enable)")
	}
	options = append(options, "删除站点 (Delete)")

	prompt := &survey.Select{
		Message: "请选择操作:",
		Options: options,
	}
	if err := survey.AskOne(prompt, &action); err != nil {
		return
	}

	switch action {
	case "返回 (Back)":
		return
	case "禁用站点 (Disable)", "启用站点 (Enable)":
		status, err := manager.ToggleSite(site.Name)
		if err != nil {
			color.Red("操作失败: %v", err)
		} else {
			if status == nginx.StatusEnabled {
				color.Green("站点已启用")
			} else {
				color.Yellow("站点已禁用")
			}
			restartSwagContainer(swagContainerName)
		}
	case "删除站点 (Delete)":
		confirm := false
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("确定要删除站点 '%s' 吗? (此操作将删除配置文件)", site.Name),
		}
		survey.AskOne(prompt, &confirm)
		if confirm {
			if err := manager.DeleteSite(site.Name); err != nil {
				color.Red("删除失败: %v", err)
			} else {
				color.Green("站点已删除")
				restartSwagContainer(swagContainerName)
			}
		}
	}
}

func restartSwagContainer(swagContainerName string) {
	color.Yellow("正在重启 SWAG 容器 (%s)...", swagContainerName)
	cli, err := docker.NewClient()
	if err != nil {
		color.Red("Docker 连接失败，无法重启容器: %v", err)
		return
	}
	if err := cli.RestartContainer(context.Background(), swagContainerName); err != nil {
		color.Red("SWAG 容器重启失败: %v", err)
	} else {
		color.Green("SWAG 容器重启成功！站点应已生效。")
	}
}
