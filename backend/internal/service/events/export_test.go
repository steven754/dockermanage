package events

// ParseSinceForTest 导出内部解析函数供外部测试包使用
func ParseSinceForTest(s string) (int64, error) {
	return parseSince(s)
}
