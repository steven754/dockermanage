// Package docker 封装 Docker Engine API 调用。
package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
)

// Client Docker 客户端包装，线程安全。
type Client struct {
	cli *client.Client
}

// New 创建 Docker 客户端。
//
// host 为空时按 FromEnv 读取 DOCKER_HOST 等环境变量，默认连接 unix socket。
// 始终开启 API 版本协商，兼容不同版本的 daemon。
func New(host string) (*Client, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}
	if host != "" {
		opts = append(opts, client.WithHost(host))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("创建 Docker 客户端失败: %w", err)
	}
	return &Client{cli: cli}, nil
}

// Raw 返回底层 Docker 客户端
func (c *Client) Raw() *client.Client { return c.cli }

// Ping 检查 daemon 连通性
func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := c.cli.Ping(ctx); err != nil {
		return fmt.Errorf("Docker daemon 不可达: %w", err)
	}
	return nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.cli == nil {
		return nil
	}
	return c.cli.Close()
}
