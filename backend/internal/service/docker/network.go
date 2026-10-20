package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"

	"github.com/steven754/dockermanage/backend/internal/model"
)

// NetworkService 网络相关操作
type NetworkService struct {
	cli *Client
}

// NewNetworkService 创建网络服务
func NewNetworkService(cli *Client) *NetworkService {
	return &NetworkService{cli: cli}
}

// List 列出网络
func (s *NetworkService) List(ctx context.Context) ([]*model.Network, error) {
	list, err := s.cli.Raw().NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, dockerErrorf("列出网络失败", err)
	}

	out := make([]*model.Network, 0, len(list))
	for _, n := range list {
		labels := n.Labels
		if labels == nil {
			labels = map[string]string{}
		}
		item := &model.Network{
			ID:             n.ID,
			Name:           n.Name,
			Driver:         n.Driver,
			Scope:          n.Scope,
			Internal:       n.Internal,
			Attachable:     n.Attachable,
			ContainerCount: len(n.Containers),
			CreatedAt:      model.FormatTime(n.Created),
			Labels:         labels,
		}
		if len(n.IPAM.Config) > 0 {
			item.IPAM = model.NetworkIPAM{
				Subnet:  n.IPAM.Config[0].Subnet,
				Gateway: n.IPAM.Config[0].Gateway,
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// Remove 删除网络
func (s *NetworkService) Remove(ctx context.Context, id string) error {
	if err := s.cli.Raw().NetworkRemove(ctx, id); err != nil {
		return dockerErrorf("删除网络失败", err)
	}
	return nil
}

// SystemService 主机与 daemon 相关操作
type SystemService struct {
	cli *Client
}

// NewSystemService 创建系统服务
func NewSystemService(cli *Client) *SystemService {
	return &SystemService{cli: cli}
}

// Ping 检查 daemon 连通性
func (s *SystemService) Ping(ctx context.Context) error {
	return s.cli.Ping(ctx)
}

// Info 返回 daemon 信息
func (s *SystemService) Info(ctx context.Context) (system.Info, error) {
	info, err := s.cli.Raw().Info(ctx)
	if err != nil {
		return system.Info{}, dockerErrorf("读取 Docker 信息失败", err)
	}
	return info, nil
}

// Version 返回 daemon 版本
func (s *SystemService) Version(ctx context.Context) (types.Version, error) {
	v, err := s.cli.Raw().ServerVersion(ctx)
	if err != nil {
		return types.Version{}, dockerErrorf("读取 Docker 版本失败", err)
	}
	return v, nil
}

// DiskUsage 返回磁盘占用统计
func (s *SystemService) DiskUsage(ctx context.Context) (types.DiskUsage, error) {
	du, err := s.cli.Raw().DiskUsage(ctx, types.DiskUsageOptions{})
	if err != nil {
		return types.DiskUsage{}, dockerErrorf("读取磁盘占用失败", err)
	}
	return du, nil
}

// Events 订阅 daemon 事件流
func (s *SystemService) Events(ctx context.Context, opts events.ListOptions) (<-chan events.Message, <-chan error) {
	return s.cli.Raw().Events(ctx, opts)
}
