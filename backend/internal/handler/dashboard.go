package handler

import (
	"context"
	"net/http"

	"github.com/docker/docker/api/types"

	"github.com/steven754/dockermanage/backend/internal/httpx"
	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/service/docker"
	"github.com/steven754/dockermanage/backend/internal/service/stats"
)

// GetDashboard 处理 GET /api/v1/dashboard
func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	ctx := r.Context()
	dashboard := &model.Dashboard{
		Containers:   model.DashboardCount{},
		Images:       model.DashboardImages{},
		Volumes:      model.DashboardVolumes{},
		Networks:     model.DashboardNetworks{},
		RecentEvents: []model.Event{},
	}

	// 主机信息
	cores := 0
	if info, err := h.system.Info(ctx); err == nil {
		cores = info.NCPU
		dashboard.Host = model.DashboardHost{
			Name:          stats.HostName(),
			DockerVersion: info.ServerVersion,
			OS:            info.OperatingSystem,
			Arch:          info.Architecture,
			CPUCores:      info.NCPU,
			UptimeSeconds: stats.Uptime(),
		}
	}
	dashboard.Host.ComposeVersion = h.composeVersion(ctx)

	// 资源占用：优先使用采样器缓存，缺失时即时采集一次
	sample, ok := h.latestSystemSample()
	if !ok {
		sample = h.collectorSample(ctx)
	}
	dashboard.Resources = h.resourcesFromSample(sample, cores)

	// 容器统计
	containers, err := h.containers.List(ctx, docker.ListFilter{})
	if err == nil {
		for _, c := range containers {
			dashboard.Containers.Total++
			switch c.State {
			case "running":
				dashboard.Containers.Running++
			case "exited", "created":
				dashboard.Containers.Stopped++
			case "dead", "removing":
				dashboard.Containers.Error++
			}
		}
	}

	// 镜像统计
	if images, err := h.images.List(ctx); err == nil {
		dashboard.Images.Total = len(images)
		for _, img := range images {
			dashboard.Images.TotalSizeBytes += img.SizeBytes
		}
	}

	// 卷统计
	if volumes, err := h.volumes.List(ctx); err == nil {
		dashboard.Volumes.Total = len(volumes)
		for _, v := range volumes {
			if v.Orphan {
				dashboard.Volumes.Orphan++
			}
		}
	}

	// 网络统计
	if networks, err := h.networks.List(ctx); err == nil {
		dashboard.Networks.Total = len(networks)
	}

	// 构建缓存
	if du, err := h.system.DiskUsage(ctx); err == nil {
		dashboard.BuildCache.SizeBytes = buildCacheSize(du.BuildCache)
	}

	// 最近事件
	if eventList, err := h.events.List(ctx, 20, "", ""); err == nil {
		for i := range eventList {
			dashboard.RecentEvents = append(dashboard.RecentEvents, *eventList[i])
		}
	}

	httpx.WriteJSON(w, http.StatusOK, dashboard)
}

// buildCacheSize 汇总构建缓存占用
func buildCacheSize(entries []*types.BuildCache) int64 {
	var total int64
	for _, e := range entries {
		if e != nil {
			total += e.Size
		}
	}
	return total
}

// resourcesFromSample 由采样构造资源占用
func (h *Handler) resourcesFromSample(s model.SystemSample, cores int) model.DashboardResource {
	memPct := 0.0
	if s.MemTotalBytes > 0 {
		memPct = float64(s.MemUsedBytes) / float64(s.MemTotalBytes) * 100
	}
	diskPct := 0.0
	if s.DiskTotalBytes > 0 {
		diskPct = float64(s.DiskUsedBytes) / float64(s.DiskTotalBytes) * 100
	}
	return model.DashboardResource{
		CPU: model.DashboardCPU{
			UsedPct: s.CPUPct,
			Cores:   cores,
		},
		Memory: model.DashboardMemory{
			UsedBytes:  s.MemUsedBytes,
			TotalBytes: s.MemTotalBytes,
			UsedPct:    memPct,
		},
		Disk: model.DashboardDisk{
			UsedBytes:  s.DiskUsedBytes,
			TotalBytes: s.DiskTotalBytes,
			UsedPct:    diskPct,
			Mountpoint: h.diskMountpoint,
		},
		Network: model.DashboardNetwork{
			RxBytesPerSec: s.NetRxBytesPerSec,
			TxBytesPerSec: s.NetTxBytesPerSec,
		},
	}
}

// latestSystemSample 返回采样器缓存的系统采样
func (h *Handler) latestSystemSample() (model.SystemSample, bool) {
	if h.sampler == nil {
		return model.SystemSample{}, false
	}
	return h.sampler.LatestSystemSample()
}

// collectorSample 即时采集一次宿主机资源
func (h *Handler) collectorSample(ctx context.Context) model.SystemSample {
	if h.collector == nil {
		return model.SystemSample{}
	}
	sample, _, _ := h.collector.Collect(ctx)
	return sample
}

// composeVersion 返回 compose 版本
func (h *Handler) composeVersion(ctx context.Context) string {
	if h.compose == nil {
		return ""
	}
	return h.compose.Version(ctx)
}
