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

var homepageCmd = &cobra.Command{
	Use:   "homepage",
	Short: "设置或清理根域名主页反代",
	Long:  "用于修改 SWAG 的 site-confs/default，使根域名（如 example.com）指向指定上游服务。",
}

var homepageSetCmd = &cobra.Command{
	Use:   "set [container_name]",
	Short: "设置根域名主页反代",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		containerName := args[0]

		domain, _ := cmd.Flags().GetString("domain")
		port, _ := cmd.Flags().GetInt("port")
		proto, _ := cmd.Flags().GetString("proto")
		keepUnderscore, _ := cmd.Flags().GetBool("keep-server-name-underscore")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		swagDir, _ := cmd.Flags().GetString("swag-dir")
		cfg := config.Config{SwagDir: swagDir}
		defaultPath, err := cfg.DefaultSiteConfPath()
		if err != nil {
			color.Red("无法定位 default 站点配置文件: %v", err)
			os.Exit(1)
		}

		editor := nginx.NewDefaultSiteEditor(defaultPath)
		res, err := editor.SetHomepage(nginx.HomepageConfig{
			Domain:                   domain,
			UpstreamApp:              containerName,
			UpstreamPort:             port,
			UpstreamProto:            proto,
			KeepServerNameUnderscore: keepUnderscore,
		}, dryRun)
		if err != nil {
			color.Red("设置主页失败: %v", err)
			os.Exit(1)
		}

		if !res.Changed {
			color.Yellow("未检测到变更，跳过写入。")
			return
		}

		if dryRun {
			color.Green("Dry-run: 已计算出需要修改的内容，但未写入文件。")
			return
		}

		if res.BackupPath != "" {
			color.Cyan("已创建备份: %s", res.BackupPath)
		}
		color.Green("已更新: %s", defaultPath)

		reloadSwagNginx(cmd)
	},
}

var homepageClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "清理根域名主页反代（恢复默认 try_files）",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		domain, _ := cmd.Flags().GetString("domain")
		restoreUnderscore, _ := cmd.Flags().GetBool("restore-server-name-underscore")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		swagDir, _ := cmd.Flags().GetString("swag-dir")
		cfg := config.Config{SwagDir: swagDir}
		defaultPath, err := cfg.DefaultSiteConfPath()
		if err != nil {
			color.Red("无法定位 default 站点配置文件: %v", err)
			os.Exit(1)
		}

		editor := nginx.NewDefaultSiteEditor(defaultPath)
		res, err := editor.ClearHomepage(domain, restoreUnderscore, dryRun)
		if err != nil {
			color.Red("清理主页失败: %v", err)
			os.Exit(1)
		}

		if !res.Changed {
			color.Yellow("未检测到变更，跳过写入。")
			return
		}

		if dryRun {
			color.Green("Dry-run: 已计算出需要修改的内容，但未写入文件。")
			return
		}

		if res.BackupPath != "" {
			color.Cyan("已创建备份: %s", res.BackupPath)
		}
		color.Green("已更新: %s", defaultPath)

		reloadSwagNginx(cmd)
	},
}

func reloadSwagNginx(cmd *cobra.Command) {
	swagContainer, _ := cmd.Flags().GetString("swag-container")
	client, err := docker.NewClient()
	if err != nil {
		color.Yellow("无法连接 Docker，跳过 Nginx reload: %v", err)
		return
	}
	color.Yellow("正在重载 SWAG (%s) Nginx...", swagContainer)
	if err := client.ReloadNginx(context.Background(), swagContainer); err != nil {
		color.Red("Nginx 重载失败: %v", err)
		color.Yellow("可尝试执行: swag-cli reload (重启容器) 以应用配置。")
		return
	}
	color.Green("Nginx 重载成功！")
}

func init() {
	homepageSetCmd.Flags().String("domain", "", "根域名 (例如 example.com)")
	homepageSetCmd.Flags().IntP("port", "p", 80, "容器内部端口")
	homepageSetCmd.Flags().String("proto", "http", "协议 (http/https)")
	homepageSetCmd.Flags().Bool("keep-server-name-underscore", false, "不修改 server_name（保持为 '_'）")
	homepageSetCmd.Flags().Bool("dry-run", false, "只计算变更，不写入文件")
	_ = homepageSetCmd.MarkFlagRequired("domain")

	homepageClearCmd.Flags().String("domain", "", "根域名（可选，仅用于记录/兼容参数）")
	homepageClearCmd.Flags().Bool("restore-server-name-underscore", true, "恢复 server_name 为 '_'")
	homepageClearCmd.Flags().Bool("dry-run", false, "只计算变更，不写入文件")

	homepageCmd.AddCommand(homepageSetCmd)
	homepageCmd.AddCommand(homepageClearCmd)
	rootCmd.AddCommand(homepageCmd)
}

