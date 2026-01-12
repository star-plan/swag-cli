package cli

import (
	"context"
	"os"
	"swag-cli/internal/config"
	"swag-cli/internal/docker"
	"swag-cli/internal/nginx"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var toggleCmd = &cobra.Command{
	Use:   "toggle [subdomain]",
	Short: "启用/禁用站点配置",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		subdomain := args[0]
		swagDir, _ := cmd.Flags().GetString("swag-dir")

		cfg := config.Config{SwagDir: swagDir}
		manager := nginx.NewManager(cfg.ProxyConfsDir())
		status, err := manager.ToggleSite(subdomain)
		if err != nil {
			color.Red("操作失败: %v", err)
			os.Exit(1)
		}

		if status == nginx.StatusEnabled {
			color.Green("站点 '%s' 已启用", subdomain)
		} else {
			color.Yellow("站点 '%s' 已禁用", subdomain)
		}

		// 触发 Nginx reload
		swagContainer, _ := cmd.Flags().GetString("swag-container")
		client, err := docker.NewClient()
		if err == nil {
			color.Yellow("正在重载 SWAG (%s) Nginx...", swagContainer)
			if err := client.ReloadNginx(context.Background(), swagContainer); err != nil {
				color.Red("Nginx 重载失败: %v", err)
			} else {
				color.Green("Nginx 重载成功！配置已生效。")
			}
		} else {
			color.Yellow("无法连接 Docker，跳过 Nginx 重载: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(toggleCmd)
}
