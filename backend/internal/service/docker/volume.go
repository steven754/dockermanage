package docker

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"

	"github.com/steven754/dockermanage/backend/internal/model"
)

// sizeBudget 卷大小统计的总耗时预算，超时后剩余卷大小记为 0
const sizeBudget = 500 * time.Millisecond

// maxWalkFiles 单个卷统计最多遍历的文件数
const maxWalkFiles = 200000

// VolumeService 数据卷相关操作
type VolumeService struct {
	cli *Client
}

// NewVolumeService 创建卷服务
func NewVolumeService(cli *Client) *VolumeService {
	return &VolumeService{cli: cli}
}

// List 列出数据卷
func (s *VolumeService) List(ctx context.Context) ([]*model.Volume, error) {
	resp, err := s.cli.Raw().VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, dockerErrorf("列出数据卷失败", err)
	}

	// 统计卷被多少容器引用
	containers, err := s.cli.Raw().ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, dockerErrorf("列出容器失败: %w", err)
	}
	refCount := map[string]int{}
	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Name != "" {
				refCount[m.Name]++
			}
		}
	}

	deadline := time.Now().Add(sizeBudget)
	out := make([]*model.Volume, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		if v == nil {
			continue
		}
		labels := v.Labels
		if labels == nil {
			labels = map[string]string{}
		}
		count := refCount[v.Name]
		item := &model.Volume{
			Name:       v.Name,
			Driver:     v.Driver,
			Scope:      v.Scope,
			Mountpoint: v.Mountpoint,
			RefCount:   count,
			Orphan:     count == 0,
			CreatedAt:  model.FormatTime(parseDockerTime(v.CreatedAt)),
			Labels:     labels,
		}
		if time.Now().Before(deadline) {
			item.SizeBytes = dirSize(v.Mountpoint, deadline)
		}
		out = append(out, item)
	}
	return out, nil
}

// Remove 删除数据卷
func (s *VolumeService) Remove(ctx context.Context, name string, force bool) error {
	if err := s.cli.Raw().VolumeRemove(ctx, name, force); err != nil {
		return dockerErrorf("删除数据卷失败", err)
	}
	return nil
}

// Prune 清理未使用的数据卷
func (s *VolumeService) Prune(ctx context.Context) (*model.PruneResult, error) {
	report, err := s.cli.Raw().VolumesPrune(ctx, filters.NewArgs())
	if err != nil {
		return nil, dockerErrorf("清理数据卷失败", err)
	}
	deleted := make([]string, 0, len(report.VolumesDeleted))
	for _, name := range report.VolumesDeleted {
		if name != "" {
			deleted = append(deleted, name)
		}
	}
	return &model.PruneResult{Deleted: deleted, SpaceReclaimedBytes: int64(report.SpaceReclaimed)}, nil
}

// dirSize 统计目录占用大小，超过时间预算或文件数上限时提前结束。
//
// Panel 通常看不到宿主机的 /var/lib/docker，此时返回 0。
func dirSize(path string, deadline time.Time) int64 {
	if path == "" {
		return 0
	}
	if _, err := os.Stat(path); err != nil {
		return 0
	}

	var total int64
	count := 0
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fs.SkipAll
		}
		if d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		count++
		if count >= maxWalkFiles {
			return fs.SkipAll
		}
		return nil
	})
	return total
}
