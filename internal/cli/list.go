package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"swag-cli/internal/config"
	"swag-cli/internal/docker"
	"swag-cli/internal/nginx"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已配置的站点及其对应的容器状态",
	Run: func(cmd *cobra.Command, args []string) {
		swagDir, _ := cmd.Flags().GetString("swag-dir")
		network, _ := cmd.Flags().GetString("network")

		// 1. 获取 Nginx 站点配置
		cfg := config.Config{SwagDir: swagDir}
		manager := nginx.NewManager(cfg.ProxyConfsDir())
		sites, err := manager.ListSites()
		if err != nil {
			color.Red("读取配置文件失败: %v", err)
			os.Exit(1)
		}

		if len(sites) == 0 {
			color.Yellow("未找到任何站点配置 (在 %s)", cfg.ProxyConfsDir())
			return
		}

		// 2. 获取 Docker 容器信息 (尝试获取，如果不成功也不阻断列表显示)
		var containerMap = make(map[string]docker.ContainerInfo)
		client, err := docker.NewClient()
		if err == nil {
			containers, err := client.ListContainersByNetwork(context.Background(), network)
			if err == nil {
				for _, c := range containers {
					// 映射 Name 和 ID
					containerMap[c.Name] = c
				}
			} else {
				color.Yellow("警告: 无法获取网络 '%s' 中的容器: %v", network, err)
			}
		} else {
			color.Yellow("警告: 无法连接 Docker: %v", err)
		}

		// 3. 显示列表
		// 格式: Subdomain | Status | Container | State | IP | Port
		fmt.Printf("%-20s | %-10s | %-20s | %-10s | %-15s | %-10s\n", "Subdomain", "Status", "Container", "State", "IP", "Port")
		fmt.Println(strings.Repeat("-", 100))

		for _, site := range sites {
			statusColor := color.New(color.FgGreen).SprintFunc()
			if site.Status == nginx.StatusDisabled {
				statusColor = color.New(color.FgRed).SprintFunc()
			}

			containerName := site.ContainerName
			containerIP := "N/A"
			containerState := "N/A"

			// 尝试关联容器信息
			if info, ok := containerMap[containerName]; ok {
				containerIP = info.IP
				containerState = info.State
				if info.State == "running" {
					containerState = color.GreenString(info.State)
				} else {
					containerState = color.RedString(info.State)
				}
				containerName = color.GreenString(containerName) // 在线
			} else if containerName != "" {
				containerName = color.RedString(containerName) // 离线或未找到
			} else {
				containerName = "Unknown"
			}

			fmt.Printf("%-20s | %-10s | %-29s | %-20s | %-15s | %-10s\n",
				site.Name,
				statusColor(site.Status),
				containerName,
				containerState,
				containerIP,
				site.ContainerPort,
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
