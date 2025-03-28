package kubeclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"bytes"
	"io"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeClient 定义Kubernetes客户端结构
type KubeClient struct {
	clientset     kubernetes.Interface // 核心API客户端
	dynamicClient dynamic.Interface    // 动态资源客户端
	restMapper    meta.RESTMapper      // 资源类型映射器
}

// NewKubeClient 创建Kubernetes客户端
func NewKubeClient(kubeconfigPath string) (*KubeClient, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build Kubernetes config: %v", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build in-cluster config: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %v", err)
	}

	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))

	return &KubeClient{
		clientset:     clientset,
		dynamicClient: dynamicClient,
		restMapper:    restMapper,
	}, nil
}

// GetPods 获取指定命名空间下的Pod列表
func (kc *KubeClient) GetPods(namespace string) ([]corev1.Pod, error) {
	pods, err := kc.clientset.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Pods: %v", err)
	}
	return pods.Items, nil
}

// GetServices 获取指定命名空间下的Service列表
func (kc *KubeClient) GetServices(namespace string) ([]corev1.Service, error) {
	services, err := kc.clientset.CoreV1().Services(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Services: %v", err)
	}
	return services.Items, nil
}

// Apply 将YAML配置应用到集群，支持多资源文档（以---分隔）
func (kc *KubeClient) Apply(yamlData []byte, namespace string) error {
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlData), 100)
	for {
		var rawObj unstructured.Unstructured
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("YAML解码失败: %v", err)
		}

		gvk := rawObj.GroupVersionKind()
		mapping, err := kc.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("资源映射失败（类型 %s）: %v", gvk, err)
		}

		// 动态判断是否需指定命名空间
		var resourceClient dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			resourceClient = kc.dynamicClient.Resource(mapping.Resource).Namespace(namespace)
		} else {
			resourceClient = kc.dynamicClient.Resource(mapping.Resource)
		}

		// 创建或更新资源
		_, err = resourceClient.Create(context.TODO(), &rawObj, v1.CreateOptions{})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				// 更新资源
				_, err = resourceClient.Update(context.TODO(), &rawObj, v1.UpdateOptions{})
				if err != nil {
					return fmt.Errorf("更新资源 %s/%s 失败: %v", gvk.Kind, rawObj.GetName(), err)
				}

			} else {
				return fmt.Errorf("创建资源 %s/%s 失败: %v", gvk.Kind, rawObj.GetName(), err)
			}
		}
	}
	return nil
}

func (kc *KubeClient) ApplyMulti(yamlDatas [][]byte, namespace string) error {
	for _, yamlData := range yamlDatas {
		if err := kc.Apply(yamlData, namespace); err != nil {
			return fmt.Errorf("应用 YAML 失败: %v", err)
		}
	}
	return nil
}

// Delete 删除 Kubernetes 资源，类似 `kubectl delete -f`
func (kc *KubeClient) Delete(yamlData []byte, namespace string) error {
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlData), 100)
	for {
		var rawObj unstructured.Unstructured
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break // 读取结束
			}
			return fmt.Errorf("解析 YAML 失败: %v", err)
		}

		// 获取 GVK（Group-Version-Kind）
		gvk := rawObj.GroupVersionKind()

		// 获取资源映射信息
		mapping, err := kc.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("无法获取资源映射: %v", err)
		}

		// 解析 namespace
		resNamespace := rawObj.GetNamespace()
		if resNamespace == "" {
			resNamespace = namespace // 如果 YAML 没有定义 namespace，则使用用户提供的
		}

		// 构造资源客户端
		resourceClient := kc.dynamicClient.Resource(mapping.Resource).Namespace(resNamespace)

		// 获取资源名称
		resourceName := rawObj.GetName()
		if resourceName == "" {
			return fmt.Errorf("资源缺少 metadata.name，无法删除")
		}

		// 执行删除操作
		err = resourceClient.Delete(context.TODO(), resourceName, v1.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				fmt.Printf("资源 %s/%s 已不存在，跳过删除\n", mapping.Resource.Resource, resourceName)
				continue
			}
			return fmt.Errorf("删除资源 %s 失败: %v", resourceName, err)
		}

		fmt.Printf("成功删除资源: %s/%s\n", mapping.Resource.Resource, resourceName)
	}
	return nil
}

func (kc *KubeClient) DeleteMulti(yamlDatas [][]byte, namespace string) error {
	for _, yamlData := range yamlDatas {
		if err := kc.Delete(yamlData, namespace); err != nil {
			return fmt.Errorf("删除 YAML 失败: %v", err)
		}
	}
	return nil
}

func (kc *KubeClient) ApplyDir(dirPath, namespace string) error {
	files, err := readYAMLFiles(dirPath)
	if err != nil {
		return fmt.Errorf("读取目录失败: %v", err)
	}

	var applyErrors []error
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			applyErrors = append(applyErrors, fmt.Errorf("读取文件 %s 失败: %v", file, err))
			continue
		}

		if err := kc.Apply(data, namespace); err != nil {
			applyErrors = append(applyErrors, fmt.Errorf("应用文件 %s 失败: %v", file, err))
		}
	}

	return kerrors.NewAggregate(applyErrors)
}

func (kc *KubeClient) DeleteDir(dirPath, namespace string) error {
	files, err := readYAMLFiles(dirPath)
	if err != nil {
		return fmt.Errorf("读取目录失败: %v", err)
	}

	// 逆序删除减少依赖冲突
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	var deleteErrors []error
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			deleteErrors = append(deleteErrors, fmt.Errorf("读取文件 %s 失败: %v", file, err))
			continue
		}

		if err := kc.Delete(data, namespace); err != nil {
			deleteErrors = append(deleteErrors, fmt.Errorf("删除文件 %s 失败: %v", file, err))
		}
	}

	return kerrors.NewAggregate(deleteErrors)
}

// 辅助函数：读取目录下所有 YAML 文件
func readYAMLFiles(dirPath string) ([]string, error) {
	var yamlFiles []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			yamlFiles = append(yamlFiles, path)
		}
		return nil
	})
	return yamlFiles, err
}
