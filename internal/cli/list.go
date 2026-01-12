package cli

import (
	"context"
	"fmt"
	"os"

	"swag-cli/internal/docker"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 SWAG 网络中的容器",
	Run: func(cmd *cobra.Command, args []string) {
		network, _ := cmd.Flags().GetString("network")

		client, err := docker.NewClient()
		if err != nil {
			color.Red("无法连接到 Docker: %v", err)
			os.Exit(1)
		}

		containers, err := client.ListContainersByNetwork(context.Background(), network)
		if err != nil {
			color.Red("获取容器列表失败: %v", err)
			os.Exit(1)
		}

		color.Green("在网络 '%s' 中发现 %d 个容器:", network, len(containers))
		fmt.Println("------------------------------------------------")
		for _, c := range containers {
			fmt.Printf("Name: %-20s IP: %s\n", c.Name, c.IP)
		}
	},
}

func init() {
	listCmd.Flags().StringP("network", "n", "swag", "指定 Docker 网络名称")
	rootCmd.AddCommand(listCmd)
}
