package model

import (
	"encoding/json"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
)

// Deploy 代表一个切片的Kubernetes接口性能控制相关参数（如 QoS、带宽、调度等）
type Deploy struct {
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
	NetworkPolicy NetworkPolicy `json:"network_policy"`

	// 特定插件使用的注解（如限速、带宽隔离）
	Annotations map[string]string `json:"annotations"`
}

// Clone 返回一个Deploy的深拷贝
func (d *Deploy) Clone() Deploy {
	// 使用json序列化和反序列化来实现深拷贝
	data, err := json.Marshal(d)
	if err != nil {
		return Deploy{} // 返回空的Deploy对象
	}

	var newDeploy Deploy
	if err := json.Unmarshal(data, &newDeploy); err != nil {
		return Deploy{} // 返回空的Deploy对象
	}

	return newDeploy
}

// 用于更新Deployment的参数
func (d *Deploy) Update(newDeploy Deploy) error {
	// 资源请求与限制
	if newDeploy.Resources != (ResourceSpec{}) {
		d.Resources = newDeploy.Resources
	}
	// 带宽限制
	if newDeploy.Bandwidth != (BandwidthSpec{}) {
		d.Bandwidth = newDeploy.Bandwidth
	}
	// 调度规则
	if newDeploy.Scheduling.SchedulerName != "" || newDeploy.Scheduling.NodeName != "" || len(newDeploy.Scheduling.NodeSelector) > 0 {
		d.Scheduling = newDeploy.Scheduling
	}
	// 网络策略
	if !newDeploy.NetworkPolicy.IsEmpty() {
		d.NetworkPolicy = newDeploy.NetworkPolicy
	}

	return nil
}

type NetworkPolicy networkingv1.NetworkPolicy

// isNetworkPolicyEmpty 严格检查网络策略是否为空。
// 只有当 NetworkPolicy 包含有意义的名称、命名空间、并且明确定义了 podSelector 和 policyTypes 时才被认为是非空。
func (policy *NetworkPolicy) IsEmpty() bool {
	// 检查元数据：名称和命名空间必须存在
	if policy.ObjectMeta.Name == "" || policy.ObjectMeta.Namespace == "" {
		return true
	}

	// 检查 Spec 字段：Spec 必须存在
	// 其零值是所有字段的零值。这里我们主要关注其中的关键字段。
	// 检查 PodSelector：必须明确存在。
	//认为一个策略必须通过选择器明确指出作用对象，
	// 而不是默认作用于所有 Pod。
	if len(policy.Spec.PodSelector.MatchLabels) == 0 && len(policy.Spec.PodSelector.MatchExpressions) == 0 {
		return true
	}

	// 检查 PolicyTypes：至少要有一个策略类型 (Ingress 或 Egress)
	// 这是 NetworkPolicy 的核心功能之一，定义了它是处理入站还是出站流量。
	if len(policy.Spec.PolicyTypes) == 0 {
		return true
	}

	if len(policy.Spec.Ingress) == 0 && len(policy.Spec.Egress) == 0 {
		return true
	}

	return false
}

// 返回Deployment的字符串表示
func (d *Deploy) String() string {
	json, err := json.Marshal(d)
	if err != nil {
		return fmt.Sprintf("Error marshaling Play: %v", err)
	}

	return string(json)
}

func (d *Deploy) Validate() error {
	if err := d.Resources.Validate(); err != nil {
		return fmt.Errorf("资源参数错误: %v", err)
	}
	if err := d.Bandwidth.Validate(); err != nil {
		return fmt.Errorf("带宽参数错误: %v", err)
	}
	if err := d.Priority.Validate(); err != nil {
		return fmt.Errorf("优先级参数错误: %v", err)
	}
	if err := d.Scheduling.Validate(); err != nil {
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

func (r *ResourceSpec) IsEmpty() bool {
	return *r == (ResourceSpec{})
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

func (b *BandwidthSpec) IsEmpty() bool {
	return *b == (BandwidthSpec{})
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

func (s *SchedulingSpec) IsEmpty() bool {
	return s.SchedulerName == "" && s.NodeName == "" && len(s.NodeSelector) == 0
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
