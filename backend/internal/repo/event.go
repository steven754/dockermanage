package repo

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// EventRecord 事件记录
type EventRecord struct {
	ID           int64
	TS           int64
	Type         string
	Action       string
	ActorID      string
	ActorName    string
	ResourceKind string
	ResourceID   string
	ResourceName string
	ExitCode     sql.NullInt64
	Details      string // JSON 文本
}

// EventRepo 事件仓储
type EventRepo struct {
	db *sql.DB
}

// Insert 写入一条事件
func (r *EventRepo) Insert(e EventRecord) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO events (ts, type, action, actor_id, actor_name, resource_kind, resource_id, resource_name, exit_code, details)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.TS, e.Type, e.Action, e.ActorID, e.ActorName, e.ResourceKind, e.ResourceID, e.ResourceName,
		nullInt(e.ExitCode), e.Details,
	)
	if err != nil {
		return 0, fmt.Errorf("写入事件失败: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取事件 ID 失败: %w", err)
	}
	return id, nil
}

// ListEventsQuery 事件查询条件
type ListEventsQuery struct {
	Limit int
	Since int64
	Type  string
}

// List 查询事件，按时间倒序
func (r *EventRepo) List(q ListEventsQuery) ([]EventRecord, error) {
	query := `SELECT id, ts, type, action, actor_id, actor_name, resource_kind, resource_id, resource_name, exit_code, details
	          FROM events WHERE 1 = 1`
	args := make([]any, 0, 3)
	if q.Since > 0 {
		query += " AND ts >= ?"
		args = append(args, q.Since)
	}
	if q.Type != "" {
		query += " AND type = ?"
		args = append(args, q.Type)
	}
	query += " ORDER BY ts DESC, id DESC"
	if q.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, q.Limit)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询事件失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]EventRecord, 0)
	for rows.Next() {
		var e EventRecord
		var actorID, actorName, resID, resName sql.NullString
		if err := rows.Scan(&e.ID, &e.TS, &e.Type, &e.Action, &actorID, &actorName,
			&e.ResourceKind, &resID, &resName, &e.ExitCode, &e.Details); err != nil {
			return nil, fmt.Errorf("扫描事件失败: %w", err)
		}
		e.ActorID = actorID.String
		e.ActorName = actorName.String
		e.ResourceID = resID.String
		e.ResourceName = resName.String
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteBefore 删除早于给定时间戳的事件
func (r *EventRepo) DeleteBefore(ts int64) (int64, error) {
	res, err := r.db.Exec("DELETE FROM events WHERE ts < ?", ts)
	if err != nil {
		return 0, fmt.Errorf("清理事件失败: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// MarshalDetails 把 details map 序列化为 JSON 文本
func MarshalDetails(details map[string]any) string {
	if len(details) == 0 {
		return "{}"
	}
	b, err := json.Marshal(details)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// UnmarshalDetails 把 JSON 文本解析回 map
func UnmarshalDetails(raw string) map[string]any {
	out := map[string]any{}
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return out
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}

// nullInt 将可空整型转为可空参数
func nullInt(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}
