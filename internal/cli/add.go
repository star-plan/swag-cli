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

var addCmd = &cobra.Command{
	Use:   "add [container_name]",
	Short: "添加一个新的反向代理配置",
	Long:  `根据指定的容器名和参数，生成 Nginx 反向代理配置文件。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 1. 获取参数
		var containerName string
		if len(args) > 0 {
			containerName = args[0]
		}

		subdomain, _ := cmd.Flags().GetString("subdomain")
		port, _ := cmd.Flags().GetInt("port")
		proto, _ := cmd.Flags().GetString("proto")
		swagDir, _ := cmd.Flags().GetString("swag-dir") // Inherited from root

		// 简单的校验 (实际场景可能需要更复杂的交互逻辑如果缺少参数)
		if containerName == "" {
			color.Red("错误: 必须指定容器名称")
			cmd.Usage()
			os.Exit(1)
		}
		if subdomain == "" {
			// 如果未指定子域名，默认使用容器名
			subdomain = containerName
		}

		// 2. 准备数据
		data := nginx.ConfigData{
			Subdomain:     subdomain,
			ContainerName: containerName,
			ContainerPort: port,
			Protocol:      proto,
		}

		// 3. 生成配置
		// 解析 proxy-confs 目录路径
		cfg := config.Config{SwagDir: swagDir}
		gen := nginx.NewGenerator(cfg.ProxyConfsDir())
		path, err := gen.GenerateConfig(data)
		if err != nil {
			color.Red("生成配置失败: %v", err)
			os.Exit(1)
		}

		color.Green("成功生成配置文件: %s", path)

		// 触发 Nginx reload
		swagContainer, _ := cmd.Flags().GetString("swag-container")
		client, err := docker.NewClient()
		if err == nil {
			color.Yellow("正在重载 SWAG (%s) Nginx...", swagContainer)
			if err := client.ReloadNginx(context.Background(), swagContainer); err != nil {
				color.Red("Nginx 重载失败: %v", err)
				// 不退出，因为配置已生成
			} else {
				color.Green("Nginx 重载成功！站点应已生效。")
			}
		} else {
			color.Yellow("无法连接 Docker，跳过 Nginx 重载: %v", err)
		}
	},
}

func init() {
	addCmd.Flags().StringP("subdomain", "s", "", "子域名 (默认为容器名)")
	addCmd.Flags().IntP("port", "p", 80, "容器内部端口")
	addCmd.Flags().String("proto", "http", "协议 (http/https)")

	rootCmd.AddCommand(addCmd)
}
