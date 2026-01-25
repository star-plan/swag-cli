package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

var configExportCmd = &cobra.Command{
	Use:   "export [file]",
	Short: "导出 swag-cli 全局配置到文件或 stdout",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stdout, _ := cmd.Flags().GetBool("stdout")
		pretty, _ := cmd.Flags().GetBool("pretty")

		if stdout && len(args) == 1 {
			color.Red("参数冲突: 使用 --stdout 时不需要提供导出文件路径")
			os.Exit(1)
		}

		cfg, err := config.Load()
		if err != nil {
			color.Red("读取配置失败: %v", err)
			os.Exit(1)
		}

		if stdout {
			var b []byte
			if pretty {
				b, err = json.MarshalIndent(cfg, "", "  ")
			} else {
				b, err = json.Marshal(cfg)
			}
			if err != nil {
				color.Red("序列化配置失败: %v", err)
				os.Exit(1)
			}
			fmt.Println(string(b))
			return
		}

		out := ""
		if len(args) == 1 {
			out = strings.TrimSpace(args[0])
		} else {
			out = defaultExportPath()
		}
		out = filepath.Clean(out)

		if err := config.ExportTo(out, cfg, pretty); err != nil {
			color.Red("导出失败: %v", err)
			os.Exit(1)
		}

		color.Green("已导出到: %s", out)
	},
}

var configImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "从文件导入 swag-cli 全局配置并写入磁盘",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		in := filepath.Clean(strings.TrimSpace(args[0]))
		yes, _ := cmd.Flags().GetBool("yes")

		newCfg, err := config.ImportFrom(in)
		if err != nil {
			color.Red("导入失败: %v", err)
			os.Exit(1)
		}

		oldCfg, _ := config.Load()
		diff := formatConfigDiff(oldCfg, newCfg)
		if diff == "" {
			color.Yellow("导入文件与当前配置一致：无变更")
		} else {
			fmt.Println("将应用以下变更：")
			fmt.Println(diff)
		}

		if !yes {
			ok, err := confirm("确认导入并覆盖本机配置吗？")
			if err != nil {
				color.Red("读取确认输入失败: %v", err)
				os.Exit(1)
			}
			if !ok {
				color.Yellow("已取消导入")
				return
			}
		}

		backupPath, err := backupCurrentConfig()
		if err != nil {
			color.Red("备份当前配置失败: %v", err)
			os.Exit(1)
		}
		if backupPath != "" {
			color.Cyan("已备份当前配置到: %s", backupPath)
		}

		if err := config.Save(newCfg); err != nil {
			color.Red("保存配置失败: %v", err)
			os.Exit(1)
		}

		color.Green("导入完成")
	},
}

func defaultExportPath() string {
	name := fmt.Sprintf("swag-cli.config.%s.json", time.Now().Format("20060102-150405"))
	return filepath.Join(".", name)
}

func formatConfigDiff(oldCfg, newCfg config.Config) string {
	keys := config.Keys()
	sort.Strings(keys)

	var lines []string
	for _, k := range keys {
		oldV, _ := config.Get(oldCfg, k)
		newV, _ := config.Get(newCfg, k)
		if strings.TrimSpace(oldV) == strings.TrimSpace(newV) {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s: %s -> %s", k, oldV, newV))
	}
	return strings.Join(lines, "\n")
}

func confirm(prompt string) (bool, error) {
	fmt.Printf("%s (y/N): ", strings.TrimSpace(prompt))
	reader := bufio.NewReader(os.Stdin)
	s, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	v := strings.ToLower(strings.TrimSpace(s))
	return v == "y" || v == "yes", nil
}

func backupCurrentConfig() (string, error) {
	p, err := config.Path()
	if err != nil {
		return "", err
	}

	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	backup := fmt.Sprintf("%s.bak-%s", p, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(backup, b, 0o644); err != nil {
		return "", err
	}
	return backup, nil
}

func init() {
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configImportCmd)
	rootCmd.AddCommand(configCmd)

	configExportCmd.Flags().Bool("stdout", false, "输出到 stdout（可用于重定向/管道）")
	configExportCmd.Flags().Bool("pretty", true, "以缩进 JSON 格式导出")
	configImportCmd.Flags().BoolP("yes", "y", false, "跳过确认，直接导入覆盖")
}
