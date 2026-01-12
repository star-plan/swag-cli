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
		// 格式: Type | Name | Target | Destination | Status | State
		fmt.Printf("%-10s | %-20s | %-10s | %-30s | %-10s | %-10s\n", "Type", "Name", "Target", "Destination", "Status", "State")
		fmt.Println(strings.Repeat("-", 110))

		for _, site := range sites {
			statusColor := color.New(color.FgGreen).SprintFunc()
			if site.Status == nginx.StatusDisabled {
				statusColor = color.New(color.FgRed).SprintFunc()
			}

			dest := site.TargetDest
			if site.TargetType == nginx.TargetContainer {
				dest = fmt.Sprintf("%s:%s", site.TargetDest, site.ContainerPort)
			} else if site.TargetType == nginx.TargetIP {
				dest = fmt.Sprintf("%s:%s", site.TargetDest, site.ContainerPort)
			}

			containerState := ""

			// 仅当目标是容器时，尝试获取容器状态
			if site.TargetType == nginx.TargetContainer {
				if info, ok := containerMap[site.TargetDest]; ok {
					containerState = info.State
					if info.State == "running" {
						containerState = color.GreenString(info.State)
						// 高亮 destination 表示在线
						dest = color.GreenString(dest)
					} else {
						containerState = color.RedString(info.State)
						dest = color.RedString(dest)
					}
				} else {
					containerState = color.RedString("Not Found")
					dest = color.RedString(dest)
				}
			} else {
				containerState = "-"
			}

			fmt.Printf("%-10s | %-20s | %-10s | %-39s | %-10s | %-20s\n",
				site.Type,
				site.Name,
				site.TargetType,
				dest,
				statusColor(site.Status),
				containerState,
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
