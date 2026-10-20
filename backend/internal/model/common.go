// Package model 定义对外 DTO，JSON tag 与 docs/api-contract.md 严格一致。
package model

import "time"

// FormatTime 将时间格式化为 RFC3339 UTC 字符串，零值返回空串。
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// FormatTimePtr 将时间格式化为 RFC3339 UTC 字符串指针，零值返回 nil。
func FormatTimePtr(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

// List 统一列表响应包装：{"items": [...]}
type List[T any] struct {
	Items []T `json:"items"`
}

// NewList 构造列表响应，保证 items 非 nil（序列化为 [] 而非 null）
func NewList[T any](items []T) List[T] {
	if items == nil {
		items = []T{}
	}
	return List[T]{Items: items}
}

// Ok 通用成功响应
type Ok struct {
	OK bool `json:"ok"`
}

// ErrorBody 错误响应体内部结构
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse 统一错误响应：{"error":{"code":"...","message":"..."}}
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}
