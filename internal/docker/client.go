package docker

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// ContainerInfo 包含我们关心的容器信息
type ContainerInfo struct {
	ID       string
	Name     string
	Image    string
	State    string // e.g., running, exited
	Status   string // e.g., Up 5 hours
	Networks []string
	IP       string
}

// Client 封装 Docker API 客户端
type Client struct {
	cli *client.Client
}

// NewClient 创建一个新的 Docker 客户端
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

// ListContainersByNetwork 列出指定网络中的所有容器
// networkName: 目标网络名称，通常是 "swag" 或用户自定义的名称
func (c *Client) ListContainersByNetwork(ctx context.Context, networkName string) ([]ContainerInfo, error) {
	// 获取所有容器
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []ContainerInfo

	for _, container := range containers {
		// 检查容器是否连接到了目标网络
		if settings, ok := container.NetworkSettings.Networks[networkName]; ok {
			name := strings.TrimPrefix(container.Names[0], "/")

			// 尝试获取别名，通常第一个别名是服务名
			// 注意：Aliases 包含 ContainerID 等，这里我们简单取 Name 或第一个有意义的 Alias
			// 实际场景中，容器名通常就是我们在 compose 中定义的 service name (如果 container_name 未指定)
			// 或者 container_name。
			// 这里我们主要使用 Name。

			info := ContainerInfo{
				ID:       container.ID,
				Name:     name,
				Image:    container.Image,
				State:    container.State,
				Status:   container.Status,
				Networks: []string{networkName},
				IP:       settings.IPAddress,
			}
			result = append(result, info)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no containers found in network '%s'", networkName)
	}

	return result, nil
}

// ReloadNginx 在指定容器中执行 nginx -s reload
func (c *Client) ReloadNginx(ctx context.Context, containerName string) error {
	// 1. 创建 Exec 配置
	execConfig := types.ExecConfig{
		Cmd:          []string{"nginx", "-s", "reload"},
		AttachStdout: true,
		AttachStderr: true,
	}

	// 2. 创建 Exec 实例
	execIDResp, err := c.cli.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}

	// 3. 启动 Exec
	resp, err := c.cli.ContainerExecAttach(ctx, execIDResp.ID, types.ExecStartCheck{})
	if err != nil {
		return fmt.Errorf("failed to attach exec: %w", err)
	}
	defer resp.Close()

	// 4. 读取输出 (可选)
	var outBuf, errBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader); err != nil {
		return fmt.Errorf("failed to read exec output: %w", err)
	}

	// 5. 检查退出代码
	execInspect, err := c.cli.ContainerExecInspect(ctx, execIDResp.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec: %w", err)
	}

	if execInspect.ExitCode != 0 {
		return fmt.Errorf("nginx reload failed (exit code %d): %s", execInspect.ExitCode, errBuf.String())
	}

	return nil
}

// Rediscover Swag Container to confirm name
func (c *Client) InspectContainer(ctx context.Context, name string) (*types.ContainerJSON, error) {
	resp, err := c.cli.ContainerInspect(ctx, name)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Exec executes a command inside a running container and returns the output
func (c *Client) Exec(ctx context.Context, containerName string, cmd []string) (string, error) {
	execConfig := types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execIDResp, err := c.cli.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	resp, err := c.cli.ContainerExecAttach(ctx, execIDResp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", fmt.Errorf("failed to attach exec: %w", err)
	}
	defer resp.Close()

	var outBuf, errBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader); err != nil {
		return "", fmt.Errorf("failed to read exec output: %w", err)
	}

	execInspect, err := c.cli.ContainerExecInspect(ctx, execIDResp.ID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect exec: %w", err)
	}

	if execInspect.ExitCode != 0 {
		return "", fmt.Errorf("command failed (exit code %d): %s | %s", execInspect.ExitCode, outBuf.String(), errBuf.String())
	}

	return outBuf.String(), nil
}

// RestartContainer 重启指定容器
func (c *Client) RestartContainer(ctx context.Context, containerName string) error {
	return c.cli.ContainerRestart(ctx, containerName, container.StopOptions{})
}
