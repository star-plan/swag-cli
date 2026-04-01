package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"swag-cli/internal/config"
	"swag-cli/internal/docker"
	"swag-cli/internal/nginx"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test [site]",
	Short: "测试已配置站点的连通性",
	Long:  `同时测试外部访问链路（域名解析）与内部访问链路（swag -> 目标容器）。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		swagDir, _ := cmd.Flags().GetString("swag-dir")
		swagContainer, _ := cmd.Flags().GetString("swag-container")
		siteFilter := ""
		if len(args) > 0 {
			siteFilter = strings.TrimSpace(args[0])
		}

		cfg := config.Config{SwagDir: swagDir}
		manager := nginx.NewManager(cfg.ProxyConfsDir())

		sites, err := manager.ListSites()
		if err != nil {
			color.Red("读取配置失败: %v", err)
			os.Exit(1)
		}

		if len(sites) == 0 {
			color.Yellow("未发现已配置站点（目录：%s）", cfg.ProxyConfsDir())
			return
		}
		sites = filterSitesByName(sites, siteFilter)
		if len(sites) == 0 {
			color.Red("未找到匹配筛选条件的站点：%s", siteFilter)
			os.Exit(1)
		}

		dockerClient, err := docker.NewClient()
		var baseDomain string
		if err != nil {
			color.Yellow("警告：Docker 连接检查失败：%v，内部链路检测可能失败。", err)
		} else {
			// Try to get URL env from swag container
			if info, err := dockerClient.InspectContainer(context.Background(), swagContainer); err == nil {
				for _, env := range info.Config.Env {
					if strings.HasPrefix(env, "URL=") {
						baseDomain = strings.TrimPrefix(env, "URL=")
						break
					}
				}
			}
		}

		fmt.Printf("开始测试 %d 个站点...\n", len(sites))
		if baseDomain != "" {
			fmt.Printf("基础域名：%s\n", baseDomain)
		}
		if siteFilter != "" {
			fmt.Printf("筛选条件：%s\n", siteFilter)
		}
		fmt.Println("")

		fmt.Printf("%-20s | %-30s | %-20s | %-25s\n", "站点", "目标", "内部检测（Swag ->）", "外部检测（Curl）")
		fmt.Println(strings.Repeat("-", 105))

		httpClient := &http.Client{
			Timeout: 5 * time.Second,
		}
		var failureDetails []string

		for _, site := range sites {
			if site.Status == nginx.StatusDisabled {
				if siteFilter != "" {
					fmt.Printf("%-20s | %-30s | %-20s | %-25s\n",
						site.Name,
						internalTargetURL(site),
						color.YellowString("已禁用"),
						color.YellowString("已禁用"),
					)
				}
				continue
			}

			// Internal Check (Swag -> Target)
			internalStatus := "-"
			targetURL := internalTargetURL(site)
			targetDisplay := targetURL
			if targetDisplay == "" {
				targetDisplay = site.TargetDest
			}
			if dockerClient != nil && (site.TargetType == nginx.TargetContainer || site.TargetType == nginx.TargetIP) {
				// Using curl -I to fetch headers only, -m 5 for timeout
				cmd := []string{"curl", "-I", "-m", "5", targetURL}
				_, err := dockerClient.Exec(context.Background(), swagContainer, cmd)
				if err == nil {
					internalStatus = color.GreenString("通过")
				} else {
					internalStatus = color.RedString("失败")
					failureDetails = append(failureDetails, fmt.Sprintf("- %s 内部检测失败（%s）：%s", site.Name, targetURL, formatCheckError(err)))
				}
			} else if site.TargetType == nginx.TargetStatic {
				internalStatus = color.CyanString("静态内容")
			}

			// External Check
			externalStatus := "-"
			if baseDomain != "" && site.Type == nginx.TypeSubdomain {
				// Construct URL: https://<site_name>.<base_domain>
				// Assuming HTTPS by default for SWAG
				// Note: site.Name for subdomain conf is just the subdomain part.
				fullURL := fmt.Sprintf("https://%s.%s", site.Name, baseDomain)

				resp, err := httpClient.Get(fullURL)
				if err == nil {
					if resp.StatusCode >= 200 && resp.StatusCode < 500 {
						externalStatus = color.GreenString("通过（%d）", resp.StatusCode)
					} else {
						externalStatus = color.RedString("失败（%d）", resp.StatusCode)
						failureDetails = append(failureDetails, fmt.Sprintf("- %s 外部检测失败（%s）：返回了异常状态码 %d", site.Name, fullURL, resp.StatusCode))
					}
					resp.Body.Close()
				} else {
					externalStatus = color.RedString("失败（不可达）")
					failureDetails = append(failureDetails, fmt.Sprintf("- %s 外部检测失败（%s）：%s", site.Name, fullURL, formatCheckError(err)))
				}
			} else if baseDomain != "" && site.Type == nginx.TypeHomepage {
				fullURL := fmt.Sprintf("https://%s", baseDomain)

				resp, err := httpClient.Get(fullURL)
				if err == nil {
					if resp.StatusCode >= 200 && resp.StatusCode < 500 {
						externalStatus = color.GreenString("通过（%d）", resp.StatusCode)
					} else {
						externalStatus = color.RedString("失败（%d）", resp.StatusCode)
						failureDetails = append(failureDetails, fmt.Sprintf("- %s 外部检测失败（%s）：返回了异常状态码 %d", site.Name, fullURL, resp.StatusCode))
					}
					resp.Body.Close()
				} else {
					externalStatus = color.RedString("失败（不可达）")
					failureDetails = append(failureDetails, fmt.Sprintf("- %s 外部检测失败（%s）：%s", site.Name, fullURL, formatCheckError(err)))
				}
			} else {
				externalStatus = color.YellowString("未配置域名")
			}

			fmt.Printf("%-20s | %-30s | %-20s | %-25s\n",
				site.Name,
				targetDisplay,
				internalStatus,
				externalStatus,
			)
		}

		if len(failureDetails) > 0 {
			fmt.Println("")
			color.Yellow("失败详情：")
			for _, detail := range failureDetails {
				fmt.Println(detail)
			}
		}
	},
}

func internalTargetURL(site nginx.SiteConfig) string {
	if strings.TrimSpace(site.TargetDest) == "" {
		return ""
	}

	proto := strings.TrimSpace(site.UpstreamProto)
	if proto == "" {
		proto = "http"
	}

	port := strings.TrimSpace(site.ContainerPort)
	if port == "" {
		port = defaultPortForProto(proto)
	}

	if port == "" {
		return fmt.Sprintf("%s://%s", proto, site.TargetDest)
	}

	return fmt.Sprintf("%s://%s:%s", proto, site.TargetDest, port)
}

func defaultPortForProto(proto string) string {
	switch strings.ToLower(strings.TrimSpace(proto)) {
	case "https":
		return "443"
	default:
		return "80"
	}
}

func filterSitesByName(sites []nginx.SiteConfig, filter string) []nginx.SiteConfig {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return sites
	}

	var filtered []nginx.SiteConfig
	for _, site := range sites {
		name := strings.ToLower(strings.TrimSpace(site.Name))
		if name == filter {
			filtered = append(filtered, site)
			continue
		}
		if filter == "homepage" && site.Type == nginx.TypeHomepage {
			filtered = append(filtered, site)
		}
	}
	return filtered
}

func formatCheckError(err error) string {
	if err == nil {
		return ""
	}

	s := strings.TrimSpace(err.Error())
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 140 {
		return s[:137] + "..."
	}
	return s
}

func init() {
	rootCmd.AddCommand(testCmd)
}
