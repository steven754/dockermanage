package model

// Settings 面板设置
type Settings struct {
	Theme                 string `json:"theme"`
	RegistryMirror        string `json:"registryMirror"`
	SampleIntervalSeconds int    `json:"sampleIntervalSeconds"`
	RetentionDays         int    `json:"retentionDays"`
	DefaultPageSize       int    `json:"defaultPageSize"`
}

// DefaultSettings 返回默认设置
func DefaultSettings() Settings {
	return Settings{
		Theme:                 "dark",
		RegistryMirror:        "",
		SampleIntervalSeconds: 10,
		RetentionDays:         7,
		DefaultPageSize:       50,
	}
}
