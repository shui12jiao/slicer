package kube

import (
	"context"
	"fmt"
	"slicer/model"

	corev1 "k8s.io/api/core/v1"

	networkingv1 "k8s.io/api/networking/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) Deploy(deploy model.Deploy, sliceID, deploymentName, namespace string) error {
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
			corev1.ResourceCPU:    resource.MustParse(deploy.Resources.CPURequest),
			corev1.ResourceMemory: resource.MustParse(deploy.Resources.MemoryRequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(deploy.Resources.CPULimit),
			corev1.ResourceMemory: resource.MustParse(deploy.Resources.MemoryLimit),
		},
	}

	// 3. 注入带宽限制（通过CNI注解）
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubernetes.io/ingress-bandwidth"] = deploy.Bandwidth.Ingress
	deployment.Spec.Template.Annotations["kubernetes.io/egress-bandwidth"] = deploy.Bandwidth.Egress

	// 4. 更新调度规则
	// 4.1 调度器名称
	deployment.Spec.Template.Spec.SchedulerName = deploy.Scheduling.SchedulerName
	// 4.2 直接节点绑定（高优先级）
	if deploy.Scheduling.NodeName != "" {
		deployment.Spec.Template.Spec.NodeName = deploy.Scheduling.NodeName
	}
	// 4.3 合并节点选择器（避免覆盖原有标签）
	for k, v := range deploy.Scheduling.NodeSelector {
		deployment.Spec.Template.Spec.NodeSelector[k] = v
	}

	// 5. 合并注解（保留系统注解）
	for k, v := range deploy.Annotations {
		deployment.Spec.Template.Annotations[k] = v
	}

	// 6. 优先级Priority处理
	if deploy.Priority != 0 { // 0表示不设置优先级
		if err := deploy.Priority.Validate(); err != nil {
			return fmt.Errorf("优先级参数错误: %v", err)
		}

		priorityClassName := deploy.Priority.ClassName(sliceID)
		oldPriorityClassName := deployment.Spec.Template.Spec.PriorityClassName
		// 若相同,直接跳过
		if oldPriorityClassName != priorityClassName {
			if oldPriorityClassName != "" {
				// 删除旧的优先级类
				if err := kc.clientset.SchedulingV1().PriorityClasses().Delete(ctx, oldPriorityClassName, metav1.DeleteOptions{}); err != nil {
					return fmt.Errorf("删除旧的优先级类失败: %v", err)
				}
			}
			// 创建新的优先级类
			priorityClass := &schedulingv1.PriorityClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: priorityClassName,
				},
				Value:         int32(deploy.Priority),
				GlobalDefault: false,
				Description:   fmt.Sprintf("Priority class for slice %s", sliceID),
				PreemptionPolicy: func() *corev1.PreemptionPolicy {
					policy := corev1.PreemptLowerPriority
					return &policy
				}(),
			}
			_, err := kc.clientset.SchedulingV1().PriorityClasses().Create(ctx, priorityClass, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("创建新的优先级类失败: %v", err)
			}
			// 更新Deployment中的优先级类名称
			deployment.Spec.Template.Spec.PriorityClassName = priorityClassName
		}
	}

	// 6. 更新Deployment
	if _, err := kc.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("更新Deployment失败: %v", err)
	}

	// 7. 创建/更新网络策略
	if err := kc.applyNetworkPolicy(&deploy.NetworkPolicy, namespace); err != nil {
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
