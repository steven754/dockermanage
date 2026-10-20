package docker

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/errdefs"

	"github.com/steven754/dockermanage/backend/internal/model"
)

// IsNotFound 判断错误是否为"资源不存在"
func IsNotFound(err error) bool {
	return errdefs.IsNotFound(err)
}

// IsConflict 判断错误是否为"状态冲突"
func IsConflict(err error) bool {
	return errdefs.IsConflict(err)
}

// normalizeImageRef 补全镜像 tag（无 tag 无 digest 时补 :latest）
func normalizeImageRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ref
	}
	if strings.Contains(ref, "@") {
		return ref
	}
	// 末段不含 ":" 说明没有 tag（同时排除了带端口的 registry 情况）
	last := ref
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		last = ref[i+1:]
	}
	if !strings.Contains(last, ":") {
		return ref + ":latest"
	}
	return ref
}

// containerFromInspect 由 inspect 结果构造容器对象
func containerFromInspect(cj types.ContainerJSON) *model.Container {
	c := &model.Container{
		ID:       cj.ID,
		Name:     strings.TrimPrefix(cj.Name, "/"),
		ImageID:  cj.Image,
		State:    "unknown",
		Status:   "unknown",
		Health:   "none",
		ExitCode: 0,
		Ports:    []model.ContainerPort{},
		Networks: []model.ContainerNetwork{},
		Mounts:   []model.ContainerMount{},
		Labels:   map[string]string{},
	}

	if cj.Config != nil {
		c.Image = cj.Config.Image
		if cj.Config.Labels != nil {
			c.Labels = cj.Config.Labels
		}
	}
	c.ComposeProject = c.Labels["com.docker.compose.project"]

	if cj.State != nil {
		c.State = cj.State.Status
		c.ExitCode = cj.State.ExitCode
		c.CreatedAt = model.FormatTime(parseDockerTime(cj.Created))
		c.StartedAt = model.FormatTime(parseDockerTime(cj.State.StartedAt))
		c.FinishedAt = model.FormatTimePtr(parseDockerTime(cj.State.FinishedAt))
		if cj.State.Health != nil {
			c.Health = strings.ToLower(cj.State.Health.Status)
		}
	}
	c.RestartCount = cj.RestartCount

	// 端口
	if cj.NetworkSettings != nil {
		for port, bindings := range cj.NetworkSettings.Ports {
			p := model.ContainerPort{
				PrivatePort: uint16(port.Int()),
				Type:        port.Proto(),
			}
			if len(bindings) > 0 {
				p.PublicPort = parseHostPort(bindings[0].HostPort)
				p.IP = bindings[0].HostIP
			}
			c.Ports = append(c.Ports, p)
		}
		// 网络
		for name, ep := range cj.NetworkSettings.Networks {
			if ep == nil {
				continue
			}
			aliases := ep.Aliases
			if aliases == nil {
				aliases = []string{}
			}
			c.Networks = append(c.Networks, model.ContainerNetwork{
				Name:    name,
				IP:      ep.IPAddress,
				Aliases: aliases,
			})
		}
	}

	// 挂载
	for _, m := range cj.Mounts {
		c.Mounts = append(c.Mounts, model.ContainerMount{
			Type:        string(m.Type),
			Name:        m.Name,
			Source:      m.Source,
			Destination: m.Destination,
			ReadOnly:    !m.RW,
		})
	}
	return c
}

// parseHostPort 解析宿主机端口字符串
func parseHostPort(s string) uint16 {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 65535 {
		return 0
	}
	return uint16(n)
}

// parseDockerTime 解析 Docker 返回的时间字符串，失败返回零值
func parseDockerTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, s); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

// dockerErrorf 构造带上下文的 Docker 错误
func dockerErrorf(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}
