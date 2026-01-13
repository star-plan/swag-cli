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
	Use:   "test",
	Short: "Tests connectivity for configured sites",
	Long:  `Tests both external accessibility (domain resolution) and internal connectivity (swag -> target container).`,
	Run: func(cmd *cobra.Command, args []string) {
		swagDir, _ := cmd.Flags().GetString("swag-dir")
		swagContainer, _ := cmd.Flags().GetString("swag-container")

		cfg := config.Config{SwagDir: swagDir}
		manager := nginx.NewManager(cfg.ProxyConfsDir())

		sites, err := manager.ListSites()
		if err != nil {
			color.Red("Failed to read configuration: %v", err)
			os.Exit(1)
		}

		if len(sites) == 0 {
			color.Yellow("No sites configured (in %s)", cfg.ProxyConfsDir())
			return
		}

		dockerClient, err := docker.NewClient()
		var baseDomain string
		if err != nil {
			color.Yellow("Warning: Docker client check failed: %v. Internal checks may fail.", err)
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

		fmt.Printf("Testing %d sites...\n", len(sites))
		if baseDomain != "" {
			fmt.Printf("Base Domain: %s\n", baseDomain)
		}
		fmt.Println("")

		fmt.Printf("%-20s | %-30s | %-20s | %-25s\n", "Name", "Target", "Internal (Swag->)", "External (Curl)")
		fmt.Println(strings.Repeat("-", 105))

		httpClient := &http.Client{
			Timeout: 5 * time.Second,
		}

		for _, site := range sites {
			if site.Status == nginx.StatusDisabled {
				continue
			}

			// Internal Check (Swag -> Target)
			internalStatus := "-"
			if dockerClient != nil && (site.TargetType == nginx.TargetContainer || site.TargetType == nginx.TargetIP) {
				// site.TargetDest is the container name or IP
				// site.ContainerPort is the port
				targetURL := fmt.Sprintf("http://%s:%s", site.TargetDest, site.ContainerPort)

				// Using curl -I to fetch headers only, -m 5 for timeout
				cmd := []string{"curl", "-I", "-m", "5", targetURL}
				_, err := dockerClient.Exec(context.Background(), swagContainer, cmd)
				if err == nil {
					internalStatus = color.GreenString("PASS")
				} else {
					internalStatus = color.RedString("FAIL")
				}
			} else if site.TargetType == nginx.TargetStatic {
				internalStatus = color.CyanString("STATIC")
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
						externalStatus = color.GreenString("PASS (%d)", resp.StatusCode)
					} else {
						externalStatus = color.RedString("FAIL (%d)", resp.StatusCode)
					}
					resp.Body.Close()
				} else {
					externalStatus = color.RedString("FAIL (Unreachable)")
				}
			} else {
				externalStatus = color.YellowString("? (No Domain)")
			}

			fmt.Printf("%-20s | %-30s | %-20s | %-25s\n",
				site.Name,
				site.TargetDest+":"+site.ContainerPort,
				internalStatus,
				externalStatus,
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
