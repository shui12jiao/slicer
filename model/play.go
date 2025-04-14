package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// Play 代表一个切片的性能控制参数
type Play struct {
	ID      primitive.ObjectID `json:"id" yaml:"id" bson:"_id,omitempty"`
	SliceID string             `json:"slice_id" yaml:"slice_id"`

	// kubernetes相关
	// QoSClass 表示服务质量等级：Guaranteed、Burstable 或 BestEffort
	QoSClass string `json:"qos_class" yaml:"qos_class"`

	// Resources 定义了 CPU 和内存的请求和限制
	Resources ResourceSpec `json:"resources" yaml:"resources"`

	// Annotations 提供了额外的注解，用于特定 CNI 插件的配置
	Annotations map[string]string `json:"annotations" yaml:"annotations"`

	// Scheduling 定义了调度策略
	Scheduling SchedulingSpec `json:"scheduling" yaml:"scheduling"`

	// Bandwidth 限制了网络带宽的上下限
	Bandwidth BandwidthSpec `json:"bandwidth" yaml:"bandwidth"`

	// NetworkPolicy 定义了网络策略，控制入站和出站流量
	NetworkPolicy networkingv1.NetworkPolicy `json:"network_policy" yaml:"network_policy"`
}

// ResourceSpec 定义了资源请求和限制
type ResourceSpec struct {
	CPURequest    string // CPU 请求，例如 "500m"
	CPULimit      string // CPU 限制，例如 "1"
	MemoryRequest string // 内存请求，例如 "512Mi"
	MemoryLimit   string // 内存限制，例如 "1Gi"
}

// BandwidthSpec 定义了网络带宽限制
type BandwidthSpec struct {
	Ingress string // 入站带宽限制，例如 "100Mbps"
	Egress  string // 出站带宽限制，例如 "200Mbps"
}

// 调度器配置
type SchedulingSpec struct {
	SchedulerName string            `json:"scheduler_name"` // 自定义调度器名称，默认 "default-scheduler"
	NodeName      string            `json:"node_name"`      // 若指定，Pod 将直接运行在此节点
	NodeSelector  map[string]string `json:"node_selector"`  // 节点标签选择器
}

// NetworkPolicySpec 定义了网络策略
type NetworkPolicySpec struct {
	Ingress []NetworkPolicyRule // 入站规则
	Egress  []NetworkPolicyRule // 出站规则
}

// NetworkPolicyRule 定义了单个网络策略规则
type NetworkPolicyRule struct {
	PodSelector map[string]string // Pod 选择器标签
	Namespace   string            // 命名空间
}
