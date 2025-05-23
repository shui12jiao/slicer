package model

import (
	"encoding/json"
	"fmt"

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

	// 优先级
	Priority Priority `json:"priority"` // 数值越大优先级越高，例如 1000

	// 保留字段, 目前不使用
	// Pod 调度规则
	Scheduling SchedulingSpec `json:"scheduling"`

	// 网络策略（前端可传入完整策略结构）
	NetworkPolicy networkingv1.NetworkPolicy `json:"network_policy"`

	// 特定插件使用的注解（如限速、带宽隔离）
	Annotations map[string]string `json:"annotations"`
}

// 用于更新play的参数
func (p *Play) Update(newPlay Play) error {
	if newPlay.SliceID != "" && newPlay.SliceID != p.SliceID {
		return fmt.Errorf("play的切片ID不匹配")
	}

	// 1. 资源请求与限制
	if newPlay.Resources != (ResourceSpec{}) {
		p.Resources = newPlay.Resources
	}
	// 2. 带宽限制
	if newPlay.Bandwidth != (BandwidthSpec{}) {
		p.Bandwidth = newPlay.Bandwidth
	}
	// 3. 调度规则
	if newPlay.Scheduling.SchedulerName != "" || newPlay.Scheduling.NodeName != "" || len(newPlay.Scheduling.NodeSelector) > 0 {
		p.Scheduling = newPlay.Scheduling
	}
	// 4. 网络策略
	if !isNetworkPolicyEmpty(newPlay.NetworkPolicy) {
		p.NetworkPolicy = newPlay.NetworkPolicy
	}

	return nil
}

// isNetworkPolicyEmpty 检查网络策略是否为空
func isNetworkPolicyEmpty(policy networkingv1.NetworkPolicy) bool {
	return policy.ObjectMeta.Name == "" && policy.ObjectMeta.Namespace == "" && len(policy.Spec.PolicyTypes) == 0
}

// 返回play的字符串表示
func (p *Play) String() string {
	json, err := json.Marshal(p)
	if err != nil {
		return fmt.Sprintf("Error marshaling Play: %v", err)
	}

	return string(json)
}

func (p *Play) Validate() error {
	if p.SliceID == "" {
		return fmt.Errorf("缺少切片ID")
	}
	if err := p.Resources.Validate(); err != nil {
		return fmt.Errorf("资源参数错误: %v", err)
	}
	if err := p.Bandwidth.Validate(); err != nil {
		return fmt.Errorf("带宽参数错误: %v", err)
	}
	if err := p.Priority.Validate(); err != nil {
		return fmt.Errorf("优先级参数错误: %v", err)
	}
	if err := p.Scheduling.Validate(); err != nil {
		return fmt.Errorf("调度参数错误: %v", err)
	}
	return nil
}

// Kubernetes中若不设置优先级,且无globalDefault为true的策略, 则默认优先级为0
type Priority int

func (p Priority) Validate() error {
	if p < 0 {
		return fmt.Errorf("优先级不能小于0")
	}
	if p > 1000000 {
		return fmt.Errorf("优先级不能大于1000000")
	}
	return nil
}

func (p Priority) ClassName(sliceID string) string {
	return fmt.Sprintf("priority-%s-%d", sliceID, p)
}

// 资源定义（CPU / 内存）
type ResourceSpec struct {
	CPURequest    string `json:"cpu_request"`    // "500m"
	CPULimit      string `json:"cpu_limit"`      // "1"
	MemoryRequest string `json:"memory_request"` // "512Mi"
	MemoryLimit   string `json:"memory_limit"`   // "1Gi"
}

func (r *ResourceSpec) Validate() error {
	if r.CPURequest == "" || r.CPULimit == "" {
		return fmt.Errorf("CPU请求和限制不能为空")
	}
	if r.MemoryRequest == "" || r.MemoryLimit == "" {
		return fmt.Errorf("内存请求和限制不能为空")
	}
	return nil
}

// 带宽配置
type BandwidthSpec struct {
	Ingress string `json:"ingress"` // 例如 "100Mbps"
	Egress  string `json:"egress"`  // 例如 "200Mbps"
}

func (b *BandwidthSpec) Validate() error {
	if b.Ingress == "" || b.Egress == "" {
		return fmt.Errorf("带宽限制不能为空")
	}
	return nil
}

// 调度器配置
type SchedulingSpec struct {
	SchedulerName string            `json:"scheduler_name"` // 自定义调度器名称，默认 "default-scheduler"
	NodeName      string            `json:"node_name"`      // 若指定，Pod 将直接运行在此节点
	NodeSelector  map[string]string `json:"node_selector"`  // 节点标签选择器
}

func (s *SchedulingSpec) Validate() error {
	if s.SchedulerName == "" {
		s.SchedulerName = "default-scheduler"
	}
	if s.NodeName != "" && len(s.NodeSelector) > 0 {
		return fmt.Errorf("不能同时指定 NodeName 和 NodeSelector")
	}
	return nil
}
