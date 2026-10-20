package repo_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/steven754/dockermanage/backend/internal/db"
	"github.com/steven754/dockermanage/backend/internal/repo"
)

// newTestStore 创建内存库并执行迁移
func newTestStore(t *testing.T) *repo.Store {
	t.Helper()

	sqlDB, err := db.Open("", true)
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.Migrate(sqlDB); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}
	return repo.New(sqlDB)
}

func TestMigrateIsIdempotent(t *testing.T) {
	sqlDB, err := db.Open("", true)
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	for i := 0; i < 3; i++ {
		if err := db.Migrate(sqlDB); err != nil {
			t.Fatalf("第 %d 次迁移失败: %v", i+1, err)
		}
	}

	var count int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&count); err != nil {
		t.Fatalf("查询迁移记录失败: %v", err)
	}
	if count == 0 {
		t.Fatal("migrations 表不应为空")
	}

	// 核心表都应存在
	for _, table := range []string{"users", "settings", "system_stats", "container_stats", "events"} {
		var name string
		err := sqlDB.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&name)
		if err != nil {
			t.Fatalf("表 %s 不存在: %v", table, err)
		}
	}
}

func TestUserRepoCRUD(t *testing.T) {
	store := newTestStore(t)

	count, err := store.Users.Count()
	if err != nil {
		t.Fatalf("统计用户失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("期望初始用户数 0，实际 %d", count)
	}

	user, err := store.Users.Create("admin", "hash-value", true)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if user.ID <= 0 {
		t.Fatal("用户 ID 应大于 0")
	}

	got, err := store.Users.GetByUsername("admin")
	if err != nil {
		t.Fatalf("按用户名查询失败: %v", err)
	}
	if got.PasswordHash != "hash-value" {
		t.Fatalf("期望密码哈希 hash-value，实际 %s", got.PasswordHash)
	}
	if !got.MustChangePassword {
		t.Fatal("must_change_password 应为 true")
	}

	byID, err := store.Users.GetByID(user.ID)
	if err != nil {
		t.Fatalf("按 ID 查询失败: %v", err)
	}
	if byID.Username != "admin" {
		t.Fatalf("期望用户名 admin，实际 %s", byID.Username)
	}

	if err := store.Users.UpdateLastLogin(user.ID); err != nil {
		t.Fatalf("更新登录时间失败: %v", err)
	}
	byID, err = store.Users.GetByID(user.ID)
	if err != nil {
		t.Fatalf("按 ID 查询失败: %v", err)
	}
	if !byID.LastLoginAt.Valid {
		t.Fatal("last_login_at 应被更新")
	}

	newVersion, err := store.Users.UpdatePassword(user.ID, "new-hash")
	if err != nil {
		t.Fatalf("更新密码失败: %v", err)
	}
	byID, err = store.Users.GetByID(user.ID)
	if err != nil {
		t.Fatalf("按 ID 查询失败: %v", err)
	}
	if byID.PasswordHash != "new-hash" {
		t.Fatalf("期望新哈希 new-hash，实际 %s", byID.PasswordHash)
	}
	if byID.MustChangePassword {
		t.Fatal("更新密码后应清除强制改密标记")
	}
	if byID.TokenVersion != newVersion {
		t.Fatalf("期望令牌版本 %d，实际 %d", newVersion, byID.TokenVersion)
	}
	if newVersion != 1 {
		t.Fatalf("首次改密后期望令牌版本自增为 1，实际 %d", newVersion)
	}
}

func TestUserRepoNotFound(t *testing.T) {
	store := newTestStore(t)

	if _, err := store.Users.GetByUsername("nobody"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("期望 ErrNotFound，实际 %v", err)
	}
	if _, err := store.Users.GetByID(999); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("期望 ErrNotFound，实际 %v", err)
	}
}

func TestSettingRepo(t *testing.T) {
	store := newTestStore(t)

	if _, err := store.Settings.Get("missing"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("期望 ErrNotFound，实际 %v", err)
	}

	if err := store.Settings.Set("theme", `"dark"`); err != nil {
		t.Fatalf("写入设置失败: %v", err)
	}
	kv, err := store.Settings.Get("theme")
	if err != nil {
		t.Fatalf("读取设置失败: %v", err)
	}
	if kv.Value != `"dark"` {
		t.Fatalf("期望值 \"dark\"，实际 %s", kv.Value)
	}

	// 覆盖写入
	if err := store.Settings.Set("theme", `"light"`); err != nil {
		t.Fatalf("覆盖写入设置失败: %v", err)
	}
	kv, _ = store.Settings.Get("theme")
	if kv.Value != `"light"` {
		t.Fatalf("期望值 \"light\"，实际 %s", kv.Value)
	}

	all, err := store.Settings.All()
	if err != nil {
		t.Fatalf("查询全部设置失败: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("期望 1 条设置，实际 %d", len(all))
	}
}

func TestSystemStatsRepo(t *testing.T) {
	store := newTestStore(t)

	rec := repo.SystemStat{
		TS:               1000,
		CPUPct:           12.5,
		MemUsedBytes:     100,
		MemTotalBytes:    200,
		DiskUsedBytes:    300,
		DiskTotalBytes:   400,
		NetRxBytesPerSec: 10,
		NetTxBytesPerSec: 20,
	}
	if err := store.SystemStats.Insert(rec); err != nil {
		t.Fatalf("写入系统采样失败: %v", err)
	}
	// 同时间戳重复写入应覆盖而非报错
	rec.CPUPct = 20
	if err := store.SystemStats.Insert(rec); err != nil {
		t.Fatalf("重复写入系统采样失败: %v", err)
	}

	list, err := store.SystemStats.ListSince(0)
	if err != nil {
		t.Fatalf("查询系统采样失败: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("期望 1 条采样，实际 %d", len(list))
	}
	if list[0].CPUPct != 20 {
		t.Fatalf("期望 CPU 20，实际 %f", list[0].CPUPct)
	}

	// 时间范围过滤
	list, err = store.SystemStats.ListSince(2000)
	if err != nil {
		t.Fatalf("查询系统采样失败: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("期望 0 条采样，实际 %d", len(list))
	}

	deleted, err := store.SystemStats.DeleteBefore(2000)
	if err != nil {
		t.Fatalf("清理系统采样失败: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("期望删除 1 行，实际 %d", deleted)
	}
}

func TestContainerStatsRepo(t *testing.T) {
	store := newTestStore(t)

	list := []repo.ContainerStat{
		{TS: 1000, ContainerID: "abc", ContainerName: "web", CPUPct: 1.5},
		{TS: 1000, ContainerID: "def", ContainerName: "db", CPUPct: 2.5},
		{TS: 1010, ContainerID: "abc", ContainerName: "web", CPUPct: 3.5},
	}
	if err := store.ContainerStats.InsertBatch(list); err != nil {
		t.Fatalf("批量写入容器采样失败: %v", err)
	}
	// 空批次不应报错
	if err := store.ContainerStats.InsertBatch(nil); err != nil {
		t.Fatalf("空批次写入失败: %v", err)
	}

	all, err := store.ContainerStats.ListSince(0, "")
	if err != nil {
		t.Fatalf("查询容器采样失败: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("期望 3 条采样，实际 %d", len(all))
	}

	filtered, err := store.ContainerStats.ListSince(0, "abc")
	if err != nil {
		t.Fatalf("按容器查询失败: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("期望 2 条采样，实际 %d", len(filtered))
	}

	deleted, err := store.ContainerStats.DeleteBefore(1005)
	if err != nil {
		t.Fatalf("清理容器采样失败: %v", err)
	}
	// ts=1000 的两条记录应被删除
	if deleted != 2 {
		t.Fatalf("期望删除 2 行，实际 %d", deleted)
	}
}

func TestEventRepo(t *testing.T) {
	store := newTestStore(t)

	id, err := store.Events.Insert(repo.EventRecord{
		TS:           1000,
		Type:         "container",
		Action:       "die",
		ActorID:      "abc",
		ActorName:    "web",
		ResourceKind: "container",
		ResourceID:   "abc",
		ResourceName: "web",
		ExitCode:     sql.NullInt64{Int64: 137, Valid: true},
		Details:      repo.MarshalDetails(map[string]any{"signal": 9}),
	})
	if err != nil {
		t.Fatalf("写入事件失败: %v", err)
	}
	if id <= 0 {
		t.Fatal("事件 ID 应大于 0")
	}

	if _, err := store.Events.Insert(repo.EventRecord{
		TS: 2000, Type: "image", Action: "pull", ResourceKind: "image",
	}); err != nil {
		t.Fatalf("写入事件失败: %v", err)
	}

	list, err := store.Events.List(repo.ListEventsQuery{Limit: 10})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("期望 2 条事件，实际 %d", len(list))
	}
	// 默认按时间倒序
	if list[0].Type != "image" {
		t.Fatalf("期望最新事件为 image，实际 %s", list[0].Type)
	}
	if !list[1].ExitCode.Valid || list[1].ExitCode.Int64 != 137 {
		t.Fatalf("期望退出码 137，实际 %v", list[1].ExitCode)
	}
	if list[1].Details == "" {
		t.Fatal("事件详情不应为空")
	}

	byType, err := store.Events.List(repo.ListEventsQuery{Type: "container"})
	if err != nil {
		t.Fatalf("按类型查询事件失败: %v", err)
	}
	if len(byType) != 1 {
		t.Fatalf("期望 1 条 container 事件，实际 %d", len(byType))
	}

	bySince, err := store.Events.List(repo.ListEventsQuery{Since: 1500})
	if err != nil {
		t.Fatalf("按时间查询事件失败: %v", err)
	}
	if len(bySince) != 1 {
		t.Fatalf("期望 1 条事件，实际 %d", len(bySince))
	}

	deleted, err := store.Events.DeleteBefore(1500)
	if err != nil {
		t.Fatalf("清理事件失败: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("期望删除 1 行，实际 %d", deleted)
	}
}

func TestMarshalUnmarshalDetails(t *testing.T) {
	details := repo.MarshalDetails(map[string]any{"signal": 9, "image": "nginx"})
	back := repo.UnmarshalDetails(details)
	if back["image"] != "nginx" {
		t.Fatalf("期望 image 为 nginx，实际 %v", back["image"])
	}

	empty := repo.UnmarshalDetails("")
	if len(empty) != 0 {
		t.Fatalf("空详情应返回空 map，实际 %v", empty)
	}
	bad := repo.UnmarshalDetails("{not json")
	if len(bad) != 0 {
		t.Fatalf("非法 JSON 应返回空 map，实际 %v", bad)
	}
}

func TestRangeStart(t *testing.T) {
	for _, key := range []string{"1h", "24h", "7d"} {
		start, _, err := repo.RangeStart(key)
		if err != nil {
			t.Fatalf("range %s 解析失败: %v", key, err)
		}
		if start <= 0 {
			t.Fatalf("range %s 的起始时间戳应大于 0", key)
		}
	}
	if _, _, err := repo.RangeStart("2h"); err == nil {
		t.Fatal("非法 range 应返回错误")
	}
}
