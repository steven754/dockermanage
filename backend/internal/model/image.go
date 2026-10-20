package model

// Image 镜像对象
type Image struct {
	ID              string   `json:"id"`
	RepoTags        []string `json:"repoTags"`
	RepoDigests     []string `json:"repoDigests"`
	SizeBytes       int64    `json:"sizeBytes"`
	SharedSizeBytes int64    `json:"sharedSizeBytes"`
	CreatedAt       string   `json:"createdAt"`
	Dangling        bool     `json:"dangling"`
	ContainerCount  int      `json:"containerCount"`
}

// PullImageRequest 拉取镜像请求体
type PullImageRequest struct {
	Image string `json:"image"`
}

// PullImageResponse 拉取镜像响应体
type PullImageResponse struct {
	OK      bool   `json:"ok"`
	ImageID string `json:"imageId"`
}

// TagImageRequest 给镜像打标签请求体
type TagImageRequest struct {
	Repo string `json:"repo"`
	Tag  string `json:"tag"`
}
