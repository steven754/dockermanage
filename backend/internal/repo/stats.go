package repo

import (
	"database/sql"
	"fmt"
	"time"
)

// SystemStat 系统级采样记录
type SystemStat struct {
	TS               int64
	CPUPct           float64
	MemUsedBytes     int64
	MemTotalBytes    int64
	DiskUsedBytes    int64
	DiskTotalBytes   int64
	NetRxBytesPerSec int64
	NetTxBytesPerSec int64
}

// ContainerStat 容器级采样记录
type ContainerStat struct {
	TS                    int64
	ContainerID           string
	ContainerName         string
	CPUPct                float64
	MemUsedBytes          int64
	MemLimitBytes         int64
	NetRxBytesPerSec      int64
	NetTxBytesPerSec      int64
	BlockReadBytesPerSec  int64
	BlockWriteBytesPerSec int64
}

// SystemStatsRepo 系统采样仓储
type SystemStatsRepo struct {
	db *sql.DB
}

// Insert 写入一条系统采样
func (r *SystemStatsRepo) Insert(s SystemStat) error {
	_, err := r.db.Exec(
		`INSERT INTO system_stats (ts, cpu_pct, mem_used_bytes, mem_total_bytes, disk_used_bytes, disk_total_bytes, net_rx_bytes_per_sec, net_tx_bytes_per_sec)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(ts) DO UPDATE SET
		   cpu_pct = excluded.cpu_pct, mem_used_bytes = excluded.mem_used_bytes, mem_total_bytes = excluded.mem_total_bytes,
		   disk_used_bytes = excluded.disk_used_bytes, disk_total_bytes = excluded.disk_total_bytes,
		   net_rx_bytes_per_sec = excluded.net_rx_bytes_per_sec, net_tx_bytes_per_sec = excluded.net_tx_bytes_per_sec`,
		s.TS, s.CPUPct, s.MemUsedBytes, s.MemTotalBytes, s.DiskUsedBytes, s.DiskTotalBytes, s.NetRxBytesPerSec, s.NetTxBytesPerSec,
	)
	if err != nil {
		return fmt.Errorf("写入系统采样失败: %w", err)
	}
	return nil
}

// ListSince 查询指定时间戳之后的系统采样
func (r *SystemStatsRepo) ListSince(since int64) ([]SystemStat, error) {
	rows, err := r.db.Query(
		`SELECT ts, cpu_pct, mem_used_bytes, mem_total_bytes, disk_used_bytes, disk_total_bytes, net_rx_bytes_per_sec, net_tx_bytes_per_sec
		 FROM system_stats WHERE ts >= ? ORDER BY ts ASC`, since)
	if err != nil {
		return nil, fmt.Errorf("查询系统采样失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]SystemStat, 0)
	for rows.Next() {
		var s SystemStat
		if err := rows.Scan(&s.TS, &s.CPUPct, &s.MemUsedBytes, &s.MemTotalBytes, &s.DiskUsedBytes,
			&s.DiskTotalBytes, &s.NetRxBytesPerSec, &s.NetTxBytesPerSec); err != nil {
			return nil, fmt.Errorf("扫描系统采样失败: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DeleteBefore 删除早于给定时间戳的采样
func (r *SystemStatsRepo) DeleteBefore(ts int64) (int64, error) {
	res, err := r.db.Exec("DELETE FROM system_stats WHERE ts < ?", ts)
	if err != nil {
		return 0, fmt.Errorf("清理系统采样失败: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ContainerStatsRepo 容器采样仓储
type ContainerStatsRepo struct {
	db *sql.DB
}

// InsertBatch 批量写入容器采样
func (r *ContainerStatsRepo) InsertBatch(list []ContainerStat) error {
	if len(list) == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(
		`INSERT INTO container_stats (ts, container_id, container_name, cpu_pct, mem_used_bytes, mem_limit_bytes,
		   net_rx_bytes_per_sec, net_tx_bytes_per_sec, block_read_bytes_per_sec, block_write_bytes_per_sec)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(ts, container_id) DO UPDATE SET
		   container_name = excluded.container_name, cpu_pct = excluded.cpu_pct, mem_used_bytes = excluded.mem_used_bytes,
		   mem_limit_bytes = excluded.mem_limit_bytes, net_rx_bytes_per_sec = excluded.net_rx_bytes_per_sec,
		   net_tx_bytes_per_sec = excluded.net_tx_bytes_per_sec,
		   block_read_bytes_per_sec = excluded.block_read_bytes_per_sec, block_write_bytes_per_sec = excluded.block_write_bytes_per_sec`)
	if err != nil {
		return fmt.Errorf("准备批量写入语句失败: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, s := range list {
		if _, err := stmt.Exec(s.TS, s.ContainerID, s.ContainerName, s.CPUPct, s.MemUsedBytes, s.MemLimitBytes,
			s.NetRxBytesPerSec, s.NetTxBytesPerSec, s.BlockReadBytesPerSec, s.BlockWriteBytesPerSec); err != nil {
			return fmt.Errorf("批量写入容器采样失败: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交容器采样失败: %w", err)
	}
	return nil
}

// ListSince 查询指定时间戳之后的容器采样，可按容器过滤
func (r *ContainerStatsRepo) ListSince(since int64, containerID string) ([]ContainerStat, error) {
	query := `SELECT ts, container_id, container_name, cpu_pct, mem_used_bytes, mem_limit_bytes,
	          net_rx_bytes_per_sec, net_tx_bytes_per_sec, block_read_bytes_per_sec, block_write_bytes_per_sec
	          FROM container_stats WHERE ts >= ?`
	args := []any{since}
	if containerID != "" {
		query += " AND container_id = ?"
		args = append(args, containerID)
	}
	query += " ORDER BY ts ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询容器采样失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]ContainerStat, 0)
	for rows.Next() {
		var s ContainerStat
		if err := rows.Scan(&s.TS, &s.ContainerID, &s.ContainerName, &s.CPUPct, &s.MemUsedBytes, &s.MemLimitBytes,
			&s.NetRxBytesPerSec, &s.NetTxBytesPerSec, &s.BlockReadBytesPerSec, &s.BlockWriteBytesPerSec); err != nil {
			return nil, fmt.Errorf("扫描容器采样失败: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DeleteBefore 删除早于给定时间戳的采样
func (r *ContainerStatsRepo) DeleteBefore(ts int64) (int64, error) {
	res, err := r.db.Exec("DELETE FROM container_stats WHERE ts < ?", ts)
	if err != nil {
		return 0, fmt.Errorf("清理容器采样失败: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// RangeStart 计算 range 字符串对应的起始时间戳
func RangeStart(rangeKey string) (int64, time.Duration, error) {
	switch rangeKey {
	case "1h":
		return time.Now().Add(-1 * time.Hour).Unix(), time.Hour, nil
	case "24h":
		return time.Now().Add(-24 * time.Hour).Unix(), 24 * time.Hour, nil
	case "7d":
		return time.Now().Add(-7 * 24 * time.Hour).Unix(), 7 * 24 * time.Hour, nil
	default:
		return 0, 0, fmt.Errorf("不支持的 range: %s", rangeKey)
	}
}
