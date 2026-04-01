package cli

import (
	"context"
	"fmt"
	"strings"
	"swag-cli/internal/docker"
)

type dockerRestarter interface {
	RestartContainer(ctx context.Context, containerName string) error
}

var newDockerClient = docker.NewClient

func restartSwagContainerByName(swagContainer string) error {
	swagContainer = strings.TrimSpace(swagContainer)
	if swagContainer == "" {
		return fmt.Errorf("未指定 SWAG 容器名称")
	}

	client, err := newDockerClient()
	if err != nil {
		return fmt.Errorf("连接 Docker 失败: %w", err)
	}

	return restartWithClient(client, swagContainer)
}

func restartWithClient(client dockerRestarter, swagContainer string) error {
	if client == nil {
		return fmt.Errorf("Docker 客户端不可用")
	}
	if err := client.RestartContainer(context.Background(), swagContainer); err != nil {
		return fmt.Errorf("重启 SWAG 容器失败: %w", err)
	}
	return nil
}
