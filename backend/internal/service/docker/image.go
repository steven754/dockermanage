package docker

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"

	"github.com/steven754/dockermanage/backend/internal/model"
)

// ImageService 镜像相关操作
type ImageService struct {
	cli *Client
}

// errInvalidTag 打标签参数非法
var errInvalidTag = errors.New("repo 与 tag 不能为空")

// pullImage 拉取镜像并返回镜像 ID
func pullImage(ctx context.Context, cli *Client, ref string) (string, error) {
	ref = normalizeImageRef(ref)
	if ref == "" {
		return "", errors.New("镜像名不能为空")
	}
	rc, err := cli.Raw().ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return "", dockerErrorf("拉取镜像失败", err)
	}
	// 必须读完整个流，否则镜像不会真正下载完成
	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()

	inspect, _, err := cli.Raw().ImageInspectWithRaw(ctx, ref)
	if err != nil {
		return "", dockerErrorf("读取镜像信息失败", err)
	}
	return inspect.ID, nil
}

// NewImageService 创建镜像服务
func NewImageService(cli *Client) *ImageService {
	return &ImageService{cli: cli}
}

// List 列出镜像
func (s *ImageService) List(ctx context.Context) ([]*model.Image, error) {
	list, err := s.cli.Raw().ImageList(ctx, image.ListOptions{All: false})
	if err != nil {
		return nil, dockerErrorf("列出镜像失败", err)
	}

	out := make([]*model.Image, 0, len(list))
	for _, img := range list {
		tags := img.RepoTags
		if tags == nil {
			tags = []string{}
		}
		digests := img.RepoDigests
		if digests == nil {
			digests = []string{}
		}
		count := 0
		if img.Containers > 0 {
			count = int(img.Containers)
		}
		out = append(out, &model.Image{
			ID:              img.ID,
			RepoTags:        tags,
			RepoDigests:     digests,
			SizeBytes:       img.Size,
			SharedSizeBytes: img.SharedSize,
			CreatedAt:       model.FormatTime(time.Unix(img.Created, 0)),
			Dangling:        len(img.RepoTags) == 0,
			ContainerCount:  count,
		})
	}
	return out, nil
}

// Pull 拉取镜像，返回镜像 ID
func (s *ImageService) Pull(ctx context.Context, ref string) (string, error) {
	return pullImage(ctx, s.cli, ref)
}

// Tag 给镜像打标签
func (s *ImageService) Tag(ctx context.Context, id, repo, tag string) error {
	repo = strings.TrimSpace(repo)
	tag = strings.TrimSpace(tag)
	if repo == "" || tag == "" {
		return errInvalidTag
	}
	target := repo + ":" + tag
	if err := s.cli.Raw().ImageTag(ctx, id, target); err != nil {
		return dockerErrorf("打标签失败", err)
	}
	return nil
}

// Remove 删除镜像
func (s *ImageService) Remove(ctx context.Context, id string, force bool) error {
	if _, err := s.cli.Raw().ImageRemove(ctx, id, image.RemoveOptions{Force: force, PruneChildren: false}); err != nil {
		return dockerErrorf("删除镜像失败", err)
	}
	return nil
}

// Prune 清理无用镜像
func (s *ImageService) Prune(ctx context.Context, dangling bool) (*model.PruneResult, error) {
	args := filters.NewArgs()
	if dangling {
		args.Add("dangling", "true")
	}
	report, err := s.cli.Raw().ImagesPrune(ctx, args)
	if err != nil {
		return nil, dockerErrorf("清理镜像失败", err)
	}
	deleted := make([]string, 0, len(report.ImagesDeleted))
	for _, d := range report.ImagesDeleted {
		if d.Deleted != "" {
			deleted = append(deleted, d.Deleted)
		}
		if d.Untagged != "" {
			deleted = append(deleted, d.Untagged)
		}
	}
	return &model.PruneResult{Deleted: deleted, SpaceReclaimedBytes: int64(report.SpaceReclaimed)}, nil
}
