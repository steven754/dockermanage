package model

// Volume 数据卷对象
type Volume struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope"`
	Mountpoint string            `json:"mountpoint"`
	SizeBytes  int64             `json:"sizeBytes"`
	RefCount   int               `json:"refCount"`
	Orphan     bool              `json:"orphan"`
	CreatedAt  string            `json:"createdAt"`
	Labels     map[string]string `json:"labels"`
}

// Network 网络对象
type Network struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Driver         string            `json:"driver"`
	Scope          string            `json:"scope"`
	Internal       bool              `json:"internal"`
	Attachable     bool              `json:"attachable"`
	IPAM           NetworkIPAM       `json:"ipam"`
	ContainerCount int               `json:"containerCount"`
	CreatedAt      string            `json:"createdAt"`
	Labels         map[string]string `json:"labels"`
}

// NetworkIPAM 网络地址分配信息
type NetworkIPAM struct {
	Subnet  string `json:"subnet"`
	Gateway string `json:"gateway"`
}
