package docker

import (
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"

	"github.com/steven754/dockermanage/backend/internal/model"
)

// NormalizeImageRefForTest 导出镜像名归一化函数供测试使用
func NormalizeImageRefForTest(ref string) string { return normalizeImageRef(ref) }

// ParseHostPortForTest 导出端口解析函数供测试使用
func ParseHostPortForTest(s string) uint16 { return parseHostPort(s) }

// ParseDockerTimeForTest 导出时间解析函数供测试使用
func ParseDockerTimeForTest(s string) time.Time { return parseDockerTime(s) }

// ContainerFromInspectForTest 导出容器转换函数供测试使用
func ContainerFromInspectForTest(cj types.ContainerJSON) *model.Container {
	return containerFromInspect(cj)
}

// HumanStatusForTest 导出状态生成函数供测试使用
func HumanStatusForTest(cj types.ContainerJSON) string { return humanStatus(cj) }

// CalculateCPUPercentForTest 导出 CPU 计算函数供测试使用
func CalculateCPUPercentForTest(cur, pre *container.CPUStats, cpus uint32) float64 {
	return calculateCPUPercent(cur, pre, cpus)
}

// ApplyFilterForTest 导出过滤函数供测试使用
func ApplyFilterForTest(list []*model.Container, f ListFilter) []*model.Container {
	return applyFilter(list, f)
}

// ContainerFromSummaryForTest 导出摘要转换函数供测试使用
func ContainerFromSummaryForTest(raw *types.Container) *model.Container {
	return containerFromSummary(raw)
}

// DeltaUintForTest 导出增量计算函数供测试使用
func DeltaUintForTest(cur, prev uint64) uint64 { return deltaUint(cur, prev) }

// IsNotFoundForTest 导出错误判定函数供测试使用
func IsNotFoundForTest(err error) bool { return IsNotFound(err) }

// IsConflictForTest 导出错误判定函数供测试使用
func IsConflictForTest(err error) bool { return IsConflict(err) }

// EndpointForTest 网络端点构造辅助
func EndpointForTest(ip string, aliases []string) *network.EndpointSettings {
	return &network.EndpointSettings{IPAddress: ip, Aliases: aliases}
}

// MountPointForTest 挂载构造辅助
func MountPointForTest(mType mount.Type, name, source, dest string, rw bool) types.MountPoint {
	return types.MountPoint{Type: mType, Name: name, Source: source, Destination: dest, RW: rw}
}
