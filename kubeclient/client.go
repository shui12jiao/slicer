package kubeclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slicer/util"
	"sort"
	"strings"

	"bytes"
	"io"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	appsv1 "k8s.io/api/apps/v1"
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
	config        util.Config          // 配置
	clientset     kubernetes.Interface // 核心API客户端
	dynamicClient dynamic.Interface    // 动态资源客户端
	restMapper    meta.RESTMapper      // 资源类型映射器
}

// NewKubeClient 创建Kubernetes客户端
func NewKubeClient(config util.Config) (kc *KubeClient, err error) {
	var kconfig *rest.Config

	// 从 kubeconfig 文件或 in-cluster 配置中创建 Kubernetes 配置
	if config.KubeconfigPath != "" {
		kconfig, err = clientcmd.BuildConfigFromFlags("", config.KubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("创建Kubernetes配置失败: %v", err)
		}
	} else {
		kconfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("创建集群内配置失败: %v", err)
		}
	}

	// clientset 用于核心API（如Pod、Service）
	clientset, err := kubernetes.NewForConfig(kconfig)
	if err != nil {
		return nil, fmt.Errorf("创建Kubernetes客户端失败: %v", err)
	}

	// dynamicClient 用于动态API（如CRD）
	dynamicClient, err := dynamic.NewForConfig(kconfig)
	if err != nil {
		return nil, fmt.Errorf("创建动态客户端失败: %v", err)
	}

	// discoveryClient 用于发现API资源
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(kconfig)
	if err != nil {
		return nil, fmt.Errorf("创建发现客户端失败: %v", err)
	}

	// restMapper 用于资源类型映射
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))

	kc = &KubeClient{
		config:        config,
		clientset:     clientset,
		dynamicClient: dynamicClient,
		restMapper:    restMapper,
	}

	// 获取所有namespaces, 作为测试
	namespaces, err := kc.GetNamespaces()
	if err != nil {
		return nil, fmt.Errorf("获取命名空间失败: %v", err)
	}
	// 检查config中的namespace是否存在, 若不存在则创建
	for _, ns := range []string{config.Namespace, config.MonitorNamespace} {
		var exist bool
		for _, namespace := range namespaces {
			if namespace.Name == ns {
				exist = true
				break
			}
		}
		if !exist {
			// 创建namespace
			namespace := &corev1.Namespace{
				ObjectMeta: v1.ObjectMeta{
					Name: ns,
				},
			}
			_, err := kc.clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, v1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("创建命名空间 %s 失败: %v", ns, err)
			}
		}
	}

	return
}

// GetNamespaces 获取所有命名空间
func (kc *KubeClient) GetNamespaces() ([]corev1.Namespace, error) {
	namespaces, err := kc.clientset.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取命名空间失败: %v", err)
	}
	return namespaces.Items, nil
}

// GetPods 获取指定命名空间下的Pod列表
func (kc *KubeClient) GetPods(namespace string, labelSelector ...string) ([]corev1.Pod, error) {
	pods, err := kc.clientset.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{
		LabelSelector: strings.Join(labelSelector, ","),
	})
	if err != nil {
		return nil, fmt.Errorf("获取Pod列表失败: %v", err)
	}
	return pods.Items, nil
}

// GetServices 获取指定命名空间下的Service列表,支持标签选择器(可选)
func (kc *KubeClient) GetServices(namespace string, labelSelector ...string) ([]corev1.Service, error) {
	// 使用标签选择器过滤Service
	serviceList, err := kc.clientset.CoreV1().Services(namespace).List(context.TODO(), v1.ListOptions{
		LabelSelector: strings.Join(labelSelector, ","),
	})
	if err != nil {
		return nil, fmt.Errorf("获取Service列表失败: %v", err)
	}
	return serviceList.Items, nil
}

func (kc *KubeClient) GetDeployments(namespace string, labelSelector ...string) ([]appsv1.Deployment, error) {
	// 使用标签选择器过滤Deployment
	deploymentList, err := kc.clientset.AppsV1().Deployments(namespace).List(context.TODO(), v1.ListOptions{
		LabelSelector: strings.Join(labelSelector, ","),
	})
	if err != nil {
		return nil, fmt.Errorf("获取Deployment列表失败: %v", err)
	}
	return deploymentList.Items, nil
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

		// 使用 Server-Side Apply
		applyOptions := v1.ApplyOptions{
			FieldManager: "application/apply-patch", // 指定字段管理器名称
			Force:        true,                      // 强制应用，必要时覆盖其他管理器的字段
		}

		// 应用资源
		_, err = resourceClient.Apply(
			context.TODO(),
			rawObj.GetName(),
			&unstructured.Unstructured{Object: rawObj.Object},
			applyOptions,
		)

		if err != nil {
			return fmt.Errorf("应用资源 %s/%s 失败: %v", gvk.Kind, rawObj.GetName(), err)
		}

		// // 创建或更新资源
		// _, err = resourceClient.Create(context.TODO(), &rawObj, v1.CreateOptions{})
		// if err != nil {
		// 	if errors.IsAlreadyExists(err) {
		// 		// 更新资源
		// 		_, err = resourceClient.Update(context.TODO(), &rawObj, v1.UpdateOptions{})
		// 		if err != nil {
		// 			return fmt.Errorf("更新资源 %s/%s 失败: %v", gvk.Kind, rawObj.GetName(), err)
		// 		}

		// 	} else {
		// 		return fmt.Errorf("创建资源 %s/%s 失败: %v", gvk.Kind, rawObj.GetName(), err)
		// 	}
		// }
	}
	return nil
}

func (kc *KubeClient) ApplyMulti(yamlDatas [][]byte, namespace string) error {
	for _, yamlData := range yamlDatas {
		if err := kc.Apply(yamlData, namespace); err != nil {
			return fmt.Errorf("应用YAML失败: %v", err)
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
			return fmt.Errorf("解析YAML失败: %v", err)
		}

		// 获取 GVK（Group-Version-Kind）
		gvk := rawObj.GroupVersionKind()

		// 获取资源映射信息
		mapping, err := kc.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("获取资源映射失败: %v", err)
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
			return fmt.Errorf("资源缺少metadata.name，无法删除")
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
			return fmt.Errorf("删除YAML失败: %v", err)
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
