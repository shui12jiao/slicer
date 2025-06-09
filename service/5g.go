package service

import (
	"log/slog"
	"slicer/db"
	"slicer/kube/value"
	"slicer/model"
	"strconv"
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

// MapSliceToValues 将切片逻辑信息转换为用于slice chart的value
func MapSliceToValues(slice model.SliceProfile) value.Slice {
	labels := map[string]any{
		"slice": slice.SliceID(), // 添加切片ID标签
	}

	// sliceProfile包含切片逻辑信息，转化为用于slice chart（upf+smf）的value
	return value.Slice{
		SMF: &value.SMF{
			CommonLabels: labels,
			Metrics: &value.SMFMetrics{
				Enabled: Ptr(slice.IsMonitored), // 是否启用监控
				ServiceMonitor: &value.SMFMetricsServiceMonitor{
					// TODO 未来改进监控配置
					Enabled: Ptr(true),
					// AdditionalLabels: labels, // TODO 可能不需要手动添加？ 待测试
				},
				ServiceScrape: &value.SMFMetricsServiceScrape{
					Enabled: Ptr(false), // 不启用VictoriaMetrics
				},
			},
			Config: &value.SMFConfig{
				Sbi: &value.SMFConfigSbi{
					Client: &value.SMFConfigSbiClient{
						Nrf: &value.SMFConfigSbiClientNrf{
							Enabled: Ptr(false), // NRF不启用
						},
						Scp: &value.SMFConfigSbiClientScp{
							Enabled: Ptr(true), // SCP启用
							Uri:     Ptr("http://scp-nscp:80"),
						},
					},
				},
				SubnetList: func() []value.SMFConfigSubnetListElem {
					subnets := make([]value.SMFConfigSubnetListElem, len(slice.AddressValue.SessionSubnets))
					for i, subnet := range slice.AddressValue.SessionSubnets {
						subnets[i] = value.SMFConfigSubnetListElem{
							Subnet:  Ptr(subnet.String()),
							Gateway: Ptr(subnet.Gateway()),
						}
					}
					return subnets
				}(),
				SliceList: []value.SMFConfigSliceListElem{
					{
						SNssai: []value.SMFConfigSliceListElemSNssaiElem{
							{
								Sst: slice.SST,
								Sd:  Ptr(slice.SD),
								Dnn: func() []string {
									dnn := make([]string, len(slice.Sessions))
									for i, session := range slice.Sessions {
										dnn[i] = session.Name
									}
									return dnn
								}(),
							},
						},
					},
				},
			},
		},
		UPF: &value.UPF{
			CommonLabels: labels,
			Metrics: &value.UPFMetrics{
				Enabled: Ptr(slice.IsMonitored), // 是否启用监控
				ServiceMonitor: &value.UPFMetricsServiceMonitor{
					// TODO 未来改进监控配置
					Enabled: Ptr(true),
					// AdditionalLabels: labels, // TODO 同上
				},
				ServiceScrape: &value.UPFMetricsServiceScrape{
					Enabled: Ptr(false), // 不启用VictoriaMetrics
				},
			},
			Config: &value.UPFConfig{
				SubnetList: func() []value.UPFConfigSubnetListElem {
					subnets := make([]value.UPFConfigSubnetListElem, len(slice.AddressValue.SessionSubnets))
					for i, subnet := range slice.AddressValue.SessionSubnets {
						subnets[i] = value.UPFConfigSubnetListElem{
							Subnet:  Ptr(subnet.String()),
							Gateway: Ptr(subnet.Gateway()),
							Dnn:     Ptr(slice.Sessions[i].Name), // 假设每个子网对应一个DNN
							Dev: func(i int) *string {
								dev := "ogstun" // 默认设备名称
								if i != 0 {
									dev += strconv.Itoa(i) // 其他设备为ogstun1, ogstun2...
								}
								return &dev
							}(i),
							Mask:      Ptr(subnet.Mask()), // 子网掩码
							CreateDev: Ptr(true),          // 创建设备
							EnableNAT: Ptr(true),          // 启用NAT
						}
					}
					return subnets
				}(),
			},
		},
	}
}

// MapCommonValues 将所有切片的公共信息转换为用于common chart的value
func MapCommonValues(slices []model.SliceProfile) value.Common {
	// TODO
	// common包含nssf，amf等chart的value，要求所有切片信息
	return value.Common{}
}

func Ptr[T any](v T) *T {
	return &v
}
