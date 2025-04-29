package kubeclient

import (
	"context"
	"fmt"
	"slicer/model"

	corev1 "k8s.io/api/core/v1"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Play 代表一个切片的性能控制参数（如 QoS、带宽、调度等）
// type Play struct {
// 	ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
// 	SliceID string             `json:"slice_id"`

// 	// QoSClass 表示服务质量等级：Guaranteed、Burstable 或 BestEffort
// 	QoSClass string `json:"qos_class"`

// 	// 资源请求与限制
// 	Resources ResourceSpec `json:"resources"`

// 	// 网络带宽限制（适用于部分 CNI）
// 	Bandwidth BandwidthSpec `json:"bandwidth"`

// 	// Pod 调度规则
// 	Scheduling SchedulingSpec `json:"scheduling"`

// 	// 网络策略（前端可传入完整策略结构）
// 	NetworkPolicy networkingv1.NetworkPolicy `json:"network_policy"`

// 	// 特定插件使用的注解（如限速、带宽隔离）
// 	Annotations map[string]string `json:"annotations"`
// }

func (kc *KubeClient) Play(play model.Play, namespace string) error {
	deploymentName := fmt.Sprintf("open5gs-upf%s", play.SliceID)
	ctx := context.Background()

	// 1. 获取现有Deployment
	deployment, err := kc.clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取Deployment失败: %v", err)
	}

	// 2. 更新资源请求/限制
	container := &deployment.Spec.Template.Spec.Containers[0]
	container.Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(play.Resources.CPURequest),
			corev1.ResourceMemory: resource.MustParse(play.Resources.MemoryRequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(play.Resources.CPULimit),
			corev1.ResourceMemory: resource.MustParse(play.Resources.MemoryLimit),
		},
	}

	// 3. 注入带宽限制（通过CNI注解）
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubernetes.io/ingress-bandwidth"] = play.Bandwidth.Ingress
	deployment.Spec.Template.Annotations["kubernetes.io/egress-bandwidth"] = play.Bandwidth.Egress

	// 4. 更新调度规则
	// 4.1 调度器名称
	deployment.Spec.Template.Spec.SchedulerName = play.Scheduling.SchedulerName
	// 4.2 直接节点绑定（高优先级）
	if play.Scheduling.NodeName != "" {
		deployment.Spec.Template.Spec.NodeName = play.Scheduling.NodeName
	}
	// 4.3 合并节点选择器（避免覆盖原有标签）
	for k, v := range play.Scheduling.NodeSelector {
		deployment.Spec.Template.Spec.NodeSelector[k] = v
	}

	// 5. 合并注解（保留系统注解）[16](@ref)
	for k, v := range play.Annotations {
		deployment.Spec.Template.Annotations[k] = v
	}

	// 6. 更新Deployment
	if _, err := kc.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("更新Deployment失败: %v", err)
	}

	// 7. 创建/更新网络策略（需独立操作）[1,2](@ref)
	if err := kc.applyNetworkPolicy(&play.NetworkPolicy, namespace); err != nil {
		return fmt.Errorf("网络策略更新失败: %v", err)
	}

	return nil
}

// 独立处理NetworkPolicy
func (kc *KubeClient) applyNetworkPolicy(np *networkingv1.NetworkPolicy, namespace string) error {
	if np == nil { // 允许空策略
		return nil
	}
	_, err := kc.clientset.NetworkingV1().NetworkPolicies(namespace).Update(
		context.Background(), np, metav1.UpdateOptions{})
	return err
}
