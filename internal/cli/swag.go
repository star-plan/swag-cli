package cli

import (
	"os"
	"strings"

	"swag-cli/internal/swagexport"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var swagCmd = &cobra.Command{
	Use:   "swag",
	Short: "管理 SWAG 容器目录相关操作",
}

var swagExportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出 SWAG 容器配置到 zip 文件",
	Run: func(cmd *cobra.Command, args []string) {
		swagDir, _ := cmd.Flags().GetString("swag-dir")
		out, _ := cmd.Flags().GetString("out")
		profileStr, _ := cmd.Flags().GetString("profile")
		includeSecrets, _ := cmd.Flags().GetBool("include-secrets")
		proxyConfOnly, _ := cmd.Flags().GetBool("proxy-conf-only")
		excludeGlobs, _ := cmd.Flags().GetStringArray("exclude-glob")

		profile := swagexport.Profile(strings.ToLower(strings.TrimSpace(profileStr)))
		switch profile {
		case swagexport.ProfileMinimal, swagexport.ProfileStandard, swagexport.ProfileFull:
		default:
			color.Red("无效 profile: %s (可选: minimal|standard|full)", profileStr)
			os.Exit(1)
		}

		if profile != swagexport.ProfileFull && includeSecrets {
			color.Yellow("提示: --include-secrets 只有在 --profile=full 时才会包含 dns-conf/keys/letsencrypt 等目录")
		}

		res, err := swagexport.Export(swagexport.Options{
			SwagDir:        swagDir,
			OutPath:        out,
			Profile:        profile,
			IncludeSecrets: includeSecrets,
			ProxyConfOnly:  proxyConfOnly,
			ExcludeGlobs:   excludeGlobs,
			Version:        cmd.Root().Version,
		})
		if err != nil {
			color.Red("导出失败: %v", err)
			os.Exit(1)
		}

		color.Green("导出完成: %s", res.OutPath)
		color.Cyan("已导出文件数: %d", res.FileCount)
		color.Cyan("归档内包含 manifest: %s", res.ManifestPath)
	},
}

func init() {
	swagCmd.AddCommand(swagExportCmd)
	rootCmd.AddCommand(swagCmd)

	swagExportCmd.Flags().String("out", "", "导出的 zip 文件路径（默认当前目录带时间戳）")
	swagExportCmd.Flags().String("profile", string(swagexport.ProfileStandard), "导出档位：minimal|standard|full")
	swagExportCmd.Flags().Bool("include-secrets", false, "允许导出敏感内容（如 dns-conf/keys/letsencrypt 等，仅 full 生效）")
	swagExportCmd.Flags().Bool("proxy-conf-only", true, "proxy-confs 仅导出 .conf/.conf.disabled，排除 sample/example")
	swagExportCmd.Flags().StringArray("exclude-glob", nil, "额外排除模式（支持 ** 通配）")
}

