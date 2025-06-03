package service

import (
	"slicer/kube/value"
	"slicer/model"
)

// Open5gs 定义Open5GS的Helm Chart相关信息
// 用于在Kubernetes集群中部署和管理Open5GS
type Open5gs struct {
	Namespace         string   // Open5GS的命名空间
	HelmCommonChart   string   // Open5GS的Common Helm Chart路径
	HelmSliceChart    string   // Open5GS的切片Helm Chart路径
	HelmReleasePrefix string   // Open5GS的Helm Release前缀
	SliceIDs          []string // 切片ID列表
}

func NewOpen5gs(namespace, HelmReleasePrefix, commonChartPath, sliceChartPath string) *Open5gs {
	return &Open5gs{
		Namespace:         namespace,
		HelmCommonChart:   commonChartPath,
		HelmSliceChart:    sliceChartPath,
		HelmReleasePrefix: HelmReleasePrefix,
		SliceIDs:          []string{},
	}
}

func (o *Open5gs) AddSlice(sliceID string) {
	// 添加切片ID到Open5GS实例中
	o.SliceIDs = append(o.SliceIDs, sliceID)
}

func (o *Open5gs) RemoveSlice(sliceID string) {
	// 从Open5GS实例中移除切片ID
	for i, id := range o.SliceIDs {
		if id == sliceID {
			o.SliceIDs = append(o.SliceIDs[:i], o.SliceIDs[i+1:]...)
			break
		}
	}
}

func (o *Open5gs) GetSliceIDs() []string {
	// 获取当前Open5GS实例中的所有切片ID
	return o.SliceIDs
}

func MapSliceToValues(slice model.SliceProfile) value.Slice {
	// TODO
	// sliceProfile包含切片逻辑信息，转化为用于slice chart（upf+smf）的value
	return value.Slice{}
}

func BuildCommonValues(slices []model.SliceProfile) value.Common {
	// TODO
	// common包含nssf，amf等chart的value，要求所有切片信息
	return value.Common{}
}
