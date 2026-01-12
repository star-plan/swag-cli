package cli

import (
	"fmt"
	"os"
	"swag-cli/internal/tui"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "swag-cli",
	Short: "SWAG (Nginx) 配置自动化助手",
	Long: `一个用于管理 SWAG (LinuxServer.io Nginx) 容器反向代理配置的 CLI 工具。
旨在简化容器发现、配置生成和站点管理流程。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 如果没有参数，默认进入交互模式 (TUI)
		confDir, _ := cmd.Flags().GetString("conf-dir")
		tui.Run(confDir)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// 这里可以定义全局 flag
	rootCmd.PersistentFlags().StringP("conf-dir", "d", "./proxy-confs", "SWAG proxy-confs 目录路径")
	rootCmd.PersistentFlags().String("swag-container", "swag", "SWAG 容器名称 (用于 reload)")
}
