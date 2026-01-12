package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"swag-cli/internal/config"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理 swag-cli 的全局配置（持久化）",
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "输出配置文件路径",
	Run: func(cmd *cobra.Command, args []string) {
		p, err := config.Path()
		if err != nil {
			color.Red("获取配置路径失败: %v", err)
			os.Exit(1)
		}
		fmt.Println(p)
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有配置项及当前值",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			color.Red("读取配置失败: %v", err)
			os.Exit(1)
		}

		keys := config.Keys()
		sort.Strings(keys)

		for _, k := range keys {
			v, _ := config.Get(cfg, k)
			fmt.Printf("%s=%s\n", k, v)
		}
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "读取指定配置项",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := strings.TrimSpace(args[0])

		cfg, err := config.Load()
		if err != nil {
			color.Red("读取配置失败: %v", err)
			os.Exit(1)
		}

		v, ok := config.Get(cfg, key)
		if !ok {
			color.Red("未知配置项: %s", key)
			os.Exit(1)
		}

		fmt.Println(v)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "设置指定配置项并写入磁盘",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := strings.TrimSpace(args[0])
		value := args[1]

		cfg, err := config.Load()
		if err != nil {
			color.Red("读取配置失败: %v", err)
			os.Exit(1)
		}

		if err := config.Set(&cfg, key, value); err != nil {
			color.Red("设置配置失败: %v", err)
			os.Exit(1)
		}

		if err := config.Save(cfg); err != nil {
			color.Red("保存配置失败: %v", err)
			os.Exit(1)
		}

		color.Green("已保存: %s=%s", key, strings.TrimSpace(value))
	},
}

func init() {
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
