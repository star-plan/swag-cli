package cli

import (
	"os"
	"swag-cli/internal/config"
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
		status, err := toggleSite(manager, subdomain)
		if err != nil {
			color.Red("操作失败: %v", err)
			os.Exit(1)
		}

		if status == nginx.StatusEnabled {
			color.Green("站点 '%s' 已启用", subdomain)
		} else {
			color.Yellow("站点 '%s' 已禁用", subdomain)
		}

		restartSwagContainer(cmd)
	},
}

func init() {
	rootCmd.AddCommand(toggleCmd)
}

type siteToggler interface {
	ToggleSite(subdomain string) (nginx.SiteStatus, error)
}

func toggleSite(manager siteToggler, subdomain string) (nginx.SiteStatus, error) {
	return manager.ToggleSite(subdomain)
}
