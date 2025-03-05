package kubeclient

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// KubeClient 封装了 Kubernetes 客户端
type KubeClient struct {
	clientset *kubernetes.Clientset
}

// NewKubeClient 创建并返回一个新的 KubeClient 实例
func NewKubeClient(kubeconfigPath string) (*KubeClient, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath != "" {
		// 使用提供的 kubeconfig 文件路径
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("构建 Kubernetes 配置失败: %v", err)
		}
	} else if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		// 使用环境变量 KUBECONFIG 指定的 kubeconfig 文件
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("构建 Kubernetes 配置失败: %v", err)
		}
	} else if home := homedir.HomeDir(); home != "" {
		// 使用默认的 kubeconfig 文件路径
		kubeconfigPath = filepath.Join(home, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("构建 Kubernetes 配置失败: %v", err)
		}
	} else {
		// 尝试使用集群内配置
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("无法找到 kubeconfig 文件，并且集群内配置失败: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("创建 Kubernetes 客户端失败: %v", err)
	}

	return &KubeClient{clientset: clientset}, nil
}

// GetPods 获取指定命名空间下的所有 Pod 信息
func (kc *KubeClient) GetPods(namespace string) {
	pods, err := kc.clientset.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatalf("获取 Pods 失败: %v", err)
	}
	fmt.Printf("命名空间 '%s' 下的 Pods:\n", namespace)
	for _, pod := range pods.Items {
		fmt.Printf("- %s\n", pod.Name)
	}
}

// GetServices 获取指定命名空间下的所有 Service 信息
func (kc *KubeClient) GetServices(namespace string) {
	services, err := kc.clientset.CoreV1().Services(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatalf("获取 Services 失败: %v", err)
	}
	fmt.Printf("命名空间 '%s' 下的 Services:\n", namespace)
	for _, svc := range services.Items {
		fmt.Printf("- %s\n", svc.Name)
	}
}
