package compose

import "github.com/steven754/dockermanage/backend/internal/model"

// ParsePsOutputForTest 导出内部解析函数供外部测试包使用
func ParsePsOutputForTest(out string) []model.ComposeService {
	return parsePsOutput(out)
}
