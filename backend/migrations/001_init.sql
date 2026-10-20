-- Dockermanage 初始表结构
-- 仅包含 v1.0 使用的五张表，notify / templates 预留到 v1.1

-- 用户（v1.0 仅单用户）
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,                -- bcrypt
    must_change_password INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL,
    last_login_at   INTEGER
);

-- 设置（kv 存储，value 为 JSON）
CREATE TABLE IF NOT EXISTS settings (
    key             TEXT PRIMARY KEY,
    value           TEXT NOT NULL,                -- JSON
    updated_at      INTEGER NOT NULL
);

-- 系统 stats 历史（采样频率：10 秒）
CREATE TABLE IF NOT EXISTS system_stats (
    ts              INTEGER NOT NULL,                -- unix timestamp
    cpu_pct          REAL NOT NULL,
    mem_used_bytes    INTEGER NOT NULL,
    mem_total_bytes   INTEGER NOT NULL,
    disk_used_bytes   INTEGER NOT NULL,
    disk_total_bytes  INTEGER NOT NULL,
    net_rx_bytes_per_sec INTEGER NOT NULL,
    net_tx_bytes_per_sec INTEGER NOT NULL,
    PRIMARY KEY (ts)
);
CREATE INDEX IF NOT EXISTS idx_system_stats_ts ON system_stats(ts);

-- 容器 stats 历史
CREATE TABLE IF NOT EXISTS container_stats (
    ts              INTEGER NOT NULL,
    container_id    TEXT NOT NULL,
    container_name  TEXT NOT NULL,
    cpu_pct         REAL NOT NULL,
    mem_used_bytes    INTEGER NOT NULL,
    mem_limit_bytes   INTEGER NOT NULL,
    net_rx_bytes_per_sec INTEGER NOT NULL,
    net_tx_bytes_per_sec INTEGER NOT NULL,
    block_read_bytes_per_sec  INTEGER NOT NULL,
    block_write_bytes_per_sec INTEGER NOT NULL,
    PRIMARY KEY (ts, container_id)
);
CREATE INDEX IF NOT EXISTS idx_container_stats_ts ON container_stats(ts);
CREATE INDEX IF NOT EXISTS idx_container_stats_container ON container_stats(container_id, ts);

-- 事件历史（保留 7 天，自动清理）
CREATE TABLE IF NOT EXISTS events (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ts              INTEGER NOT NULL,
    type            TEXT NOT NULL,                  -- container|image|volume|network|daemon
    action          TEXT NOT NULL,
    actor_id        TEXT,
    actor_name      TEXT,
    resource_kind   TEXT NOT NULL,
    resource_id     TEXT,
    resource_name   TEXT,
    exit_code       INTEGER,
    details         TEXT                            -- JSON
);
CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
