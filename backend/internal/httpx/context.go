package httpx

import "context"

// contextKey 私有上下文键类型，避免与其他包冲突
type contextKey string

// userIDKey 用户 ID 的上下文键
const userIDKey contextKey = "panel.user_id"

// WithUserID 把用户 ID 写入上下文
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext 从上下文读取用户 ID
func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}
