package service

import (
	"fmt"
	"log/slog"
	"slicer/kube/value"
	"slicer/model"
	"strconv"
)

// Open5gs 定义Open5GS的Helm Chart相关信息
// 用于在Kubernetes集群中部署和管理Open5GS
type Open5gs struct {
	service *Service // 父Service引用，用于访问Service方法

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

func (o *Open5gs) GenerateValues(sliceID string) (sliceVals map[string]value.Slice, commonVal value.Common, err error) {
	sliceVals = make(map[string]value.Slice)

	// 获取所有切片信息
	slices, err := o.service.Store.ListSlice()
	if err != nil {
		slog.Error("获取切片信息失败", "error", err)
		return
	}
	commonVal = o.MapCommonValues(slices)
	if sliceID == "" { // 如果没有指定切片ID，则生成所有切片的Values
		for _, slice := range slices {
			sliceVals[slice.SliceID()] = o.MapSliceToValues(slice)
		}
	} else {
		for _, slice := range slices {
			if slice.SliceID() == sliceID {
				sliceVals[sliceID] = o.MapSliceToValues(slice)
				break // 找到指定的切片ID后退出循环
			}
		}
	}
	slog.Debug("生成Open5GS的Values", "common", commonVal.ToMap(), "slices", value.ToMap(sliceVals))
	return
}

// MapSliceToValues 将切片逻辑信息转换为用于slice chart的value
func (o *Open5gs) MapSliceToValues(slice *model.SliceProfile) value.Slice {
	labels := map[string]any{
		"slice": slice.SliceID(), // 添加切片ID标签
	}

	var um *value.UPFMetrics
	var sm *value.SMFMetrics
	// 如果切片启用了监控，则获取监控信息
	if slice.MonitorRef != nil {
		// 查找监控信息
		monitor, err := o.service.Store.GetMonitor(slice.MonitorRef.Hex())
		if err != nil {
			// 监控信息获取失败，不使用监控配置
			slog.Error("获取监控信息失败，不配置监控", "monitorID", slice.MonitorRef.Hex(), "error", err)
		} else {
			// 转换监控信息为UPF和SMF的Metrics
			um, sm = o.MapMetricsValue(monitor)
		}
	}

	// sliceProfile包含切片逻辑信息，转化为用于slice chart（upf+smf）的value
	return value.Slice{
		SMF: &value.SMF{
			CommonLabels: labels,
			Metrics:      sm, // SMF的监控配置
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
			// UPF采用的tag不同，注意！
			Image: &value.UPFImage{
				Registry:    Ptr("crpi-sut5dyyu9y5gqtfq.cn-shanghai.personal.cr.aliyuncs.com"),
				Repository:  Ptr("sminggg/open5gs"),
				Tag:         Ptr("2.7.0-upf-metrics-v2"), // Open5GS的Docker镜像版本
				Digest:      Ptr(""),                     // 镜像的Digest，如果
				PullPolicy:  Ptr("IfNotPresent"),         // 镜像拉取策略
				PullSecrets: []any{},                     // 镜像拉取密钥
				Debug:       Ptr(false),                  // 是否启用调试日志
			}, // UPF的Docker镜像地址
			Command: []any{
				"/open5gs/install/bin/open5gs-upfd", // UPF的启动命令
			},
			Args: []any{
				"-c",
				"/opt/open5gs/etc/open5gs/upf.yaml", // UPF的配置文件路径
			},
			CommonLabels: labels,
			Metrics:      um, // UPF的监控配置
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
func (o *Open5gs) MapCommonValues(slices []*model.SliceProfile) value.Common {
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
				PlmnList: func(slices []*model.SliceProfile) []value.AMFConfigPlmnListElem {
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
				NsiList: func(slices []*model.SliceProfile) []value.NSSFConfigNsiListElem {
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

// MapMetricsValue 将监控信息转换为UPF和SMF的Metrics配置
// 如果监控信息为空，则使用默认值
// 监控信息包含监控间隔和超时设置
func (o *Open5gs) MapMetricsValue(m *model.Monitor) (um *value.UPFMetrics, sm *value.SMFMetrics) {
	if m == nil {
		slog.Warn("监控信息为空，使用默认值")
		// 如果监控信息为空，则使用默认值
		um, sm = o.MapMetricsValue(&model.Monitor{})
		return
	}

	um = &value.UPFMetrics{
		Enabled: Ptr(true),
		ServiceMonitor: &value.UPFMetricsServiceMonitor{
			Enabled: Ptr(true),
		},
		ServiceScrape: &value.UPFMetricsServiceScrape{
			Enabled: Ptr(false), // 不启用VictoriaMetrics
		},
	}
	sm = &value.SMFMetrics{
		Enabled: Ptr(true),
		ServiceMonitor: &value.SMFMetricsServiceMonitor{
			Enabled: Ptr(true),
		},
		ServiceScrape: &value.SMFMetricsServiceScrape{
			Enabled: Ptr(false), // 不启用VictoriaMetrics
		},
	}

	if m.MonitoringInterval.IntervalSecs > 0 {
		internal := fmt.Sprintf("%ds", m.MonitoringInterval.IntervalSecs)
		um.ServiceMonitor.Interval = &internal
		sm.ServiceMonitor.Interval = &internal
	}
	if m.Timeout > 0 {
		timeout := fmt.Sprintf("%ds", int(m.Timeout.Seconds()))
		um.ServiceMonitor.ScrapeTimeout = &timeout
		sm.ServiceMonitor.ScrapeTimeout = &timeout
	}

	return
}

// 用于对字面量类型的值进行指针化
func Ptr[T any](v T) *T {
	return &v
}
