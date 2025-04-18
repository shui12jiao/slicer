package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	networkingv1 "k8s.io/api/networking/v1"
)

// Play 代表一个切片的性能控制参数（如 QoS、带宽、调度等）
type Play struct {
	ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SliceID string             `json:"slice_id"`

	// 资源请求与限制
	Resources ResourceSpec `json:"resources"`

	// 网络带宽限制（适用于部分 CNI）
	Bandwidth BandwidthSpec `json:"bandwidth"`

	// Pod 调度规则
	Scheduling SchedulingSpec `json:"scheduling"`

	// 网络策略（前端可传入完整策略结构）
	NetworkPolicy networkingv1.NetworkPolicy `json:"network_policy"`

	// 特定插件使用的注解（如限速、带宽隔离）
	Annotations map[string]string `json:"annotations"`
}

// 资源定义（CPU / 内存）
type ResourceSpec struct {
	CPURequest    string `json:"cpu_request"`    // "500m"
	CPULimit      string `json:"cpu_limit"`      // "1"
	MemoryRequest string `json:"memory_request"` // "512Mi"
	MemoryLimit   string `json:"memory_limit"`   // "1Gi"
}

// 带宽配置
type BandwidthSpec struct {
	Ingress string `json:"ingress"` // 例如 "100Mbps"
	Egress  string `json:"egress"`  // 例如 "200Mbps"
}

// 调度器配置
type SchedulingSpec struct {
	SchedulerName string            `json:"scheduler_name"` // 自定义调度器名称，默认 "default-scheduler"
	NodeName      string            `json:"node_name"`      // 若指定，Pod 将直接运行在此节点
	NodeSelector  map[string]string `json:"node_selector"`  // 节点标签选择器
}
