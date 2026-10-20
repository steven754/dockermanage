package model

// Event 事件对象
type Event struct {
	ID           int64          `json:"id"`
	TS           string         `json:"ts"`
	Type         string         `json:"type"`
	Action       string         `json:"action"`
	ActorID      string         `json:"actorId"`
	ActorName    string         `json:"actorName"`
	ResourceID   string         `json:"resourceId"`
	ResourceName string         `json:"resourceName"`
	ExitCode     *int           `json:"exitCode"`
	Details      map[string]any `json:"details"`
}

// ListEventsRequest 事件列表查询条件
type ListEventsRequest struct {
	Limit int    `query:"limit"`
	Since string `query:"since"`
	Type  string `query:"type"`
}
