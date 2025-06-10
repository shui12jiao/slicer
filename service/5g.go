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

	// PLMN
	MCC string // 移动国家代码
	MNC string // 移动网络代码
	// TODO 其他配置?
}

func NewOpen5gs(namespace, HelmReleasePrefix, commonChartPath, sliceChartPath string) *Open5gs {
	return &Open5gs{
		Namespace:         namespace,
		HelmCommonChart:   commonChartPath,
		HelmSliceChart:    sliceChartPath,
		HelmReleasePrefix: HelmReleasePrefix,

		MCC: "999", // 默认MCC
		MNC: "70",  // 默认MNC
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

	commonVal = o.MapCommonValues(slices)
	if !commonOnly { // 如果不是仅获取公共值，则需要获取每个切片的值
		for _, slice := range slices {
			sliceVals[slice.SliceID()] = o.MapSliceToValues(slice)
		}
		slog.Info("生成Open5GS的Values", "sliceCount", len(sliceVals), "common", commonVal)
	} else {
		slog.Info("生成Open5GS的公共Values", "common", commonVal)
	}
	return
}

// MapSliceToValues 将切片逻辑信息转换为用于slice chart的value
func (o *Open5gs) MapSliceToValues(slice model.SliceProfile) value.Slice {
	labels := map[string]any{
		"slice": slice.SliceID(), // 添加切片ID标签
	}

	// sliceProfile包含切片逻辑信息，转化为用于slice chart（upf+smf）的value
	return value.Slice{
		SMF: &value.SMF{
			CommonLabels: labels,
			Metrics: &value.SMFMetrics{
				Enabled: &slice.IsMonitored, // 是否启用监控
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
								Sd:  &slice.SD,
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
				Enabled: &slice.IsMonitored, // 是否启用监控
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
							Dnn:     &slice.Sessions[i].Name, // 假设每个子网对应一个DNN
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
func (o *Open5gs) MapCommonValues(slices []model.SliceProfile) value.Common {
	// common包含nssf，amf等chart的value，要求所有切片信息
	return value.Common{
		AMF: &value.AMF{
			Metrics: &value.AMFMetrics{
				Enabled: Ptr(true), // 启用AMF监控
				ServiceMonitor: &value.AMFMetricsServiceMonitor{
					Enabled: Ptr(true), // 启用ServiceMonitor
				},
				ServiceScrape: &value.AMFMetricsServiceScrape{
					Enabled: Ptr(false), // 不启用VictoriaMetrics
				},
			},
			Config: &value.AMFConfig{
				Sbi: &value.AMFConfigSbi{
					Client: &value.AMFConfigSbiClient{
						Nrf: &value.AMFConfigSbiClientNrf{
							Enabled: Ptr(false), // NRF不启用
						},
						Scp: &value.AMFConfigSbiClientScp{
							Enabled: Ptr(true), // SCP启用
						},
					},
				},
				GuamiList: []value.AMFConfigGuamiListElem{
					{
						PlmnId: &value.AMFConfigGuamiListElemPlmnId{
							Mcc: &o.MCC, // 默认MCC
							Mnc: &o.MNC, // 默认MNC
						},
						AmfId: &value.AMFConfigGuamiListElemAmfId{
							Region: Ptr(int(2)), // 默认Region
							Set:    Ptr(int(1)), // 默认Set
						},
					},
				},
				TaiList: []value.AMFConfigTaiListElem{
					{
						PlmnId: &value.AMFConfigTaiListElemPlmnId{
							Mcc: &o.MCC, // 默认MCC
							Mnc: &o.MNC, // 默认MNC
						},
						Tac: []int{1, 2, 3}, // 默认TAC
					},
				},
				PlmnList: func(slices []model.SliceProfile) []value.AMFConfigPlmnListElem {
					snssai := make([]value.AMFConfigPlmnListElemSNssaiElem, len(slices))
					for i, slice := range slices {
						snssai[i] = value.AMFConfigPlmnListElemSNssaiElem{
							Sd:  &slice.SD,
							Sst: &slice.SST,
						}
					}
					return []value.AMFConfigPlmnListElem{
						{
							PlmnId: &value.AMFConfigPlmnListElemPlmnId{
								Mcc: &o.MCC, // 默认MCC
								Mnc: &o.MNC, // 默认MNC
							},
							SNssai: snssai,
						},
					}
				}(slices),
				NetworkName: Ptr("Open5GS"), // 网络名称
			},
		},
		AUSF: &value.AUSF{
			Config: &value.AUSFConfig{
				Sbi: &value.AUSFConfigSbi{
					Client: &value.AUSFConfigSbiClient{
						Nrf: &value.AUSFConfigSbiClientNrf{
							Enabled: Ptr(false), // NRF不启用
						},
						Scp: &value.AUSFConfigSbiClientScp{
							Enabled: Ptr(true), // SCP启用
						},
					},
				},
			},
		},
		BSF: &value.BSF{
			Config: &value.BSFConfig{
				Sbi: &value.BSFConfigSbi{
					Client: &value.BSFConfigSbiClient{
						Nrf: &value.BSFConfigSbiClientNrf{
							Enabled: Ptr(false), // NRF不启用
						},
						Scp: &value.BSFConfigSbiClientScp{
							Enabled: Ptr(true), // SCP启用
						},
					},
				},
			},
		},
		NRF: &value.NRF{
			Config: &value.NRFConfig{
				ServingList: []value.NRFConfigServingListElem{
					{
						PlmnId: &value.NRFConfigServingListElemPlmnId{
							Mcc: &o.MCC, // 默认MCC
							Mnc: &o.MNC, // 默认MNC
						},
					},
				},
			},
		},
		NSSF: &value.NSSF{
			Config: &value.NSSFConfig{
				Sbi: &value.NSSFConfigSbi{
					Client: &value.NSSFConfigSbiClient{
						Nrf: &value.NSSFConfigSbiClientNrf{
							Enabled: Ptr(false), // NRF不启用
						},
						Scp: &value.NSSFConfigSbiClientScp{
							Enabled: Ptr(true), // SCP启用
						},
					},
				},
				NsiList: func(slices []model.SliceProfile) []value.NSSFConfigNsiListElem {
					nsiList := make([]value.NSSFConfigNsiListElem, len(slices))
					for i, slice := range slices {
						nsiList[i] = value.NSSFConfigNsiListElem{
							Sd:  &slice.SD,
							Sst: &slice.SST,
							Uri: Ptr("http://nrf-nnrf:80"),
						}
					}
					return nsiList
				}(slices),
			},
		},
		PCF: &value.PCF{
			Metrics: &value.PCFMetrics{
				Enabled: Ptr(true), // 启用PCF监控
				ServiceMonitor: &value.PCFMetricsServiceMonitor{
					Enabled: Ptr(true), // 启用ServiceMonitor
				},
				ServiceScrape: &value.PCFMetricsServiceScrape{
					Enabled: Ptr(false), // 不启用VictoriaMetrics
				},
			},
			Config: &value.PCFConfig{
				Sbi: &value.PCFConfigSbi{
					Client: &value.PCFConfigSbiClient{
						Nrf: &value.PCFConfigSbiClientNrf{
							Enabled: Ptr(false), // NRF不启用
						},
						Scp: &value.PCFConfigSbiClientScp{
							Enabled: Ptr(true), // SCP启用
						},
					},
				},
			},
		},
		SCP: &value.SCP{
			Config: &value.SCPConfig{
				Sbi: &value.SCPConfigSbi{
					Client: &value.SCPConfigSbiClient{
						Nrf: &value.SCPConfigSbiClientNrf{
							Enabled: Ptr(true), // NRF启用
						},
					},
				},
			},
		},
		UDM: &value.UDM{
			Config: &value.UDMConfig{
				Sbi: &value.UDMConfigSbi{
					Client: &value.UDMConfigSbiClient{
						Nrf: &value.UDMConfigSbiClientNrf{
							Enabled: Ptr(false), // NRF不启用
						},
						Scp: &value.UDMConfigSbiClientScp{
							Enabled: Ptr(true), // SCP启用
						},
					},
				},
			},
		},
		UDR: &value.UDR{
			Config: &value.UDRConfig{
				Sbi: &value.UDRConfigSbi{
					Client: &value.UDRConfigSbiClient{
						Nrf: &value.UDRConfigSbiClientNrf{
							Enabled: Ptr(false), // NRF不启用
						},
						Scp: &value.UDRConfigSbiClientScp{
							Enabled: Ptr(true), // SCP启用
						},
					},
				},
			},
		},
		WebUI: &value.Open5GSWebUI{}, // Open5GS Web UI配置
	}
}

// 用于对字面量类型的值进行指针化
func Ptr[T any](v T) *T {
	return &v
}
