package cli

import (
	"context"
	"os"
	"swag-cli/internal/docker"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "重启 SWAG 容器",
	Long:  `重启指定的 SWAG 容器以应用新的配置。`,
	Run: func(cmd *cobra.Command, args []string) {
		swagContainer, _ := cmd.Flags().GetString("swag-container")

		if swagContainer == "" {
			color.Red("未指定 SWAG 容器名称，请使用 --swag-container 标志或检查配置文件")
			os.Exit(1)
		}

		color.Blue("正在重启 SWAG 容器: %s ...", swagContainer)

		client, err := docker.NewClient()
		if err != nil {
			color.Red("连接 Docker 失败: %v", err)
			os.Exit(1)
		}

		// 使用 RestartContainer 而不是 ReloadNginx，因为在某些情况下（如新增子域）需要重启容器才能生效
		err = client.RestartContainer(context.Background(), swagContainer)
		if err != nil {
			color.Red("重启 SWAG 容器失败: %v", err)
			os.Exit(1)
		}

		color.Green("SWAG 容器已成功重启")
	},
}

func init() {
	rootCmd.AddCommand(reloadCmd)
}
