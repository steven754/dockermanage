// Package events 监听 Docker daemon 事件流，持久化并实时广播。
package events

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"

	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/service/broker"
)

// Service 事件服务
type Service struct {
	system    EventSource
	repo      *repo.EventRepo
	broker    *broker.Broker[model.Event]
	retention time.Duration

	mu   sync.Mutex
	last map[string]time.Time
}

// EventSource Docker 事件流来源
type EventSource interface {
	Events(ctx context.Context, opts events.ListOptions) (<-chan events.Message, <-chan error)
}

// New 创建事件服务
func New(source EventSource, eventRepo *repo.EventRepo, retention time.Duration) *Service {
	if retention <= 0 {
		retention = 7 * 24 * time.Hour
	}
	return &Service{
		system:    source,
		repo:      eventRepo,
		broker:    broker.New[model.Event](),
		retention: retention,
		last:      map[string]time.Time{},
	}
}

// Broker 返回广播中心
func (s *Service) Broker() *broker.Broker[model.Event] { return s.broker }

// Run 监听事件流直到 ctx 取消，断线后自动重连
func (s *Service) Run(ctx context.Context) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			slog.Info("事件监听已停止")
			return
		}

		msgs, errs := s.system.Events(ctx, events.ListOptions{
			Filters: filters.NewArgs(),
		})

		s.consume(ctx, msgs, errs)

		if ctx.Err() != nil {
			return
		}
		slog.Warn("Docker 事件流中断，准备重连", "backoff", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// consume 消费事件流
func (s *Service) consume(ctx context.Context, msgs <-chan events.Message, errs <-chan error) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errs:
			if err != nil {
				slog.Error("读取 Docker 事件失败", "error", err)
			}
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			s.handle(msg)
		}
	}
}

// handle 处理单条事件
func (s *Service) handle(msg events.Message) {
	event, ok := Convert(msg)
	if !ok {
		return
	}

	// 去重：同一 actor + action 在 1 秒内只记录一次
	key := event.ActorID + "|" + event.Type + "|" + event.Action
	now := time.Now()
	s.mu.Lock()
	if last, exists := s.last[key]; exists && now.Sub(last) < time.Second {
		s.mu.Unlock()
		return
	}
	s.last[key] = now
	// 清理过期记录，避免内存无限增长
	for k, t := range s.last {
		if now.Sub(t) > time.Minute {
			delete(s.last, k)
		}
	}
	s.mu.Unlock()

	record := repo.EventRecord{
		TS:           eventTime(msg),
		Type:         event.Type,
		Action:       event.Action,
		ActorID:      event.ActorID,
		ActorName:    event.ActorName,
		ResourceKind: event.Type,
		ResourceID:   event.ResourceID,
		ResourceName: event.ResourceName,
		Details:      repo.MarshalDetails(event.Details),
	}
	if event.ExitCode != nil {
		record.ExitCode = sqlNullInt(int64(*event.ExitCode))
	}

	id, err := s.repo.Insert(record)
	if err != nil {
		slog.Error("写入事件失败", "error", err)
		return
	}
	event.ID = id
	// 广播前补齐落库后的 ID 与时间戳
	event.TS = model.FormatTime(time.Unix(record.TS, 0))
	s.broker.Publish(*event)
}

// List 查询历史事件
func (s *Service) List(ctx context.Context, limit int, since string, eventType string) ([]*model.Event, error) {
	q := repo.ListEventsQuery{Limit: limit, Type: eventType}
	if since != "" {
		ts, err := parseSince(since)
		if err != nil {
			return nil, err
		}
		q.Since = ts
	}

	rows, err := s.repo.List(q)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Event, 0, len(rows))
	for i := range rows {
		out = append(out, toModel(&rows[i]))
	}
	return out, nil
}

// Cleanup 清理超过保留期的事件
func (s *Service) Cleanup(ctx context.Context) {
	cutoff := time.Now().Add(-s.retention).Unix()
	if n, err := s.repo.DeleteBefore(cutoff); err != nil {
		slog.Error("清理事件失败", "error", err)
	} else if n > 0 {
		slog.Info("已清理历史事件", "rows", n)
	}
}

// toModel 将仓库记录转换为模型
func toModel(r *repo.EventRecord) *model.Event {
	e := &model.Event{
		ID:           r.ID,
		TS:           model.FormatTime(time.Unix(r.TS, 0)),
		Type:         r.Type,
		Action:       r.Action,
		ActorID:      r.ActorID,
		ActorName:    r.ActorName,
		ResourceID:   r.ResourceID,
		ResourceName: r.ResourceName,
		Details:      repo.UnmarshalDetails(r.Details),
	}
	if r.ExitCode.Valid {
		code := int(r.ExitCode.Int64)
		e.ExitCode = &code
	}
	return e
}

// Convert 将 Docker 事件转换为面板事件
func Convert(msg events.Message) (*model.Event, bool) {
	eventType := string(msg.Type)
	if eventType == "" {
		return nil, false
	}
	action := string(msg.Action)
	if action == "" {
		return nil, false
	}

	details := map[string]any{}
	for k, v := range msg.Actor.Attributes {
		details[k] = v
	}

	name := msg.Actor.Attributes["name"]
	image := msg.Actor.Attributes["image"]

	e := &model.Event{
		TS:           model.FormatTime(time.Unix(eventTime(msg), 0)),
		Type:         eventType,
		Action:       action,
		ActorID:      msg.Actor.ID,
		ActorName:    name,
		ResourceID:   msg.Actor.ID,
		ResourceName: name,
		Details:      details,
	}
	if e.ActorName == "" {
		e.ActorName = msg.Actor.ID
	}
	if e.ResourceName == "" {
		e.ResourceName = msg.Actor.ID
	}

	// 容器退出码
	if code, ok := msg.Actor.Attributes["exitCode"]; ok && code != "" {
		if v, err := strconv.Atoi(code); err == nil {
			e.ExitCode = &v
			e.Details["exitCode"] = v
		}
	}
	if image != "" {
		e.Details["image"] = image
	}
	return e, true
}

// eventTime 取事件时间
func eventTime(msg events.Message) int64 {
	if msg.TimeNano > 0 {
		return msg.TimeNano / int64(time.Second)
	}
	if msg.Time > 0 {
		return msg.Time
	}
	return time.Now().Unix()
}

// parseSince 解析 since 查询参数（RFC3339 或 unix 秒）
func parseSince(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Unix(), nil
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v, nil
	}
	return 0, ErrInvalidSince
}

// ErrInvalidSince since 参数非法
var ErrInvalidSince = errors.New("since 参数格式非法，期望 RFC3339 或 unix 时间戳")

// sqlNullInt 构造可空整型
func sqlNullInt(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: true}
}
