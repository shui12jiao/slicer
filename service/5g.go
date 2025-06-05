package service

import (
	"log/slog"
	"slicer/db"
	"slicer/kube/value"
	"slicer/model"
)

// Open5gs 定义Open5GS的Helm Chart相关信息
// 用于在Kubernetes集群中部署和管理Open5GS
type Open5gs struct {
	Namespace         string // Open5GS的命名空间
	HelmCommonChart   string // Open5GS的Common Helm Chart路径
	HelmSliceChart    string // Open5GS的切片Helm Chart路径
	HelmReleasePrefix string // Open5GS的Helm Release前缀
}

func NewOpen5gs(namespace, HelmReleasePrefix, commonChartPath, sliceChartPath string) *Open5gs {
	return &Open5gs{
		Namespace:         namespace,
		HelmCommonChart:   commonChartPath,
		HelmSliceChart:    sliceChartPath,
		HelmReleasePrefix: HelmReleasePrefix,
	}
}

func (o *Open5gs) GenerateValues(store db.Store, commonOnly bool) (sliceVals map[string]value.Slice, commonVal value.Common, err error) {
	sliceVals = make(map[string]value.Slice)

	// 获取所有切片信息
	slices, err := store.ListSlice()
	if err != nil {
		slog.Error("获取切片信息失败", "error", err)
		return
	}

	commonVal = MapCommonValues(slices)
	if !commonOnly { // 如果不是仅获取公共值，则需要获取每个切片的值
		for _, slice := range slices {
			sliceVals[slice.SliceID()] = MapSliceToValues(slice)
		}
		slog.Info("生成Open5GS的Values", "sliceCount", len(sliceVals), "common", commonVal)
	} else {
		slog.Info("生成Open5GS的公共Values", "common", commonVal)
	}
	return
}

func MapSliceToValues(slice model.SliceProfile) value.Slice {
	// sliceProfile包含切片逻辑信息，转化为用于slice chart（upf+smf）的value
	return value.Slice{
		SMF: &value.SMF{
			Config: &value.SMFConfig{
				SubnetList: []value.SMFConfigSubnetListElem{
					// TODO
				},
			},
		},
		UPF: &value.UPF{},
	}
}

func MapCommonValues(slices []model.SliceProfile) value.Common {
	// TODO
	// common包含nssf，amf等chart的value，要求所有切片信息
	return value.Common{}
}
