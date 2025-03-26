package kubeclient

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slicer/util"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery/cached/memory"
	fakediscovery "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
)

// newFakeKubeClient 构造一个使用 fake client 的 KubeClient 实例
func newFakeKubeClient(t *testing.T) *KubeClient {
	scheme := runtime.NewScheme()
	// 创建 fake clientset，并设置 fake discovery 可返回所需的 APIResourceList
	fakeCS := fakeclientset.NewSimpleClientset()
	fakeDisc, ok := fakeCS.Discovery().(*fakediscovery.FakeDiscovery)
	require.True(t, ok, "获取 fake discovery 失败")
	fakeDisc.Fake.Resources = []*v1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []v1.APIResource{
				{
					Name:       "configmaps",
					Kind:       "ConfigMap",
					Namespaced: true,
					Verbs:      []string{"get", "list", "create", "update", "delete"},
				},
				{
					Name:       "pods",
					Kind:       "Pod",
					Namespaced: true,
					Verbs:      []string{"get", "list"},
				},
				{
					Name:       "services",
					Kind:       "Service",
					Namespaced: true,
					Verbs:      []string{"get", "list"},
				},
			},
		},
	}

	// 创建 fake dynamic client
	fakeDyn := dynamicfake.NewSimpleDynamicClient(scheme)
	// 构造 restMapper
	rm := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(fakeDisc))

	return &KubeClient{
		clientset:     fakeCS,
		dynamicClient: fakeDyn,
		restMapper:    rm,
	}
}

func TestGetPods(t *testing.T) {
	kc := newFakeKubeClient(t)
	// 创建一个 Pod 对象并写入 fake clientset
	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}
	_, err := kc.clientset.CoreV1().Pods("default").Create(context.TODO(), pod, v1.CreateOptions{})
	require.NoError(t, err)

	pods, err := kc.GetPods("default")
	require.NoError(t, err)
	assert.Equal(t, 1, len(pods))
	assert.Equal(t, "test-pod", pods[0].Name)
}

func TestGetServices(t *testing.T) {
	kc := newFakeKubeClient(t)
	svc := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
	}
	_, err := kc.clientset.CoreV1().Services("default").Create(context.TODO(), svc, v1.CreateOptions{})
	require.NoError(t, err)

	svcs, err := kc.GetServices("default")
	require.NoError(t, err)
	assert.Equal(t, 1, len(svcs))
	assert.Equal(t, "test-svc", svcs[0].Name)
}

func TestApply(t *testing.T) {
	kc := newFakeKubeClient(t)
	yamlData := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value
`)
	err := kc.Apply(yamlData, "default")
	require.NoError(t, err)

	// 验证在 fake dynamic client 中能获取到该 ConfigMap
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	obj, err := kc.dynamicClient.Resource(gvr).Namespace("default").Get(context.TODO(), "test-cm", v1.GetOptions{})
	require.NoError(t, err)

	data, found, err := unstructured.NestedStringMap(obj.Object, "data")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, map[string]string{"key": "value"}, data)
}

func TestDelete(t *testing.T) {
	kc := newFakeKubeClient(t)
	// 预先创建 ConfigMap 对象
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	cm := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test-cm",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key": "value",
			},
		},
	}
	_, err := kc.dynamicClient.Resource(gvr).Namespace("default").Create(context.TODO(), cm, v1.CreateOptions{})
	require.NoError(t, err)

	// 使用 YAML 删除该 ConfigMap（注意 YAML 中未指定 data）
	yamlData := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
`)
	err = kc.Delete(yamlData, "default")
	require.NoError(t, err)

	// 验证已删除
	_, err = kc.dynamicClient.Resource(gvr).Namespace("default").Get(context.TODO(), "test-cm", v1.GetOptions{})
	assert.Error(t, err)
}

func TestApplyMulti(t *testing.T) {
	kc := newFakeKubeClient(t)
	yaml1 := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
data:
  key1: value1
`)
	yaml2 := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm2
data:
  key2: value2
`)
	yamls := [][]byte{yaml1, yaml2}
	err := kc.ApplyMulti(yamls, "default")
	require.NoError(t, err)

	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	obj1, err := kc.dynamicClient.Resource(gvr).Namespace("default").Get(context.TODO(), "cm1", v1.GetOptions{})
	require.NoError(t, err)
	obj2, err := kc.dynamicClient.Resource(gvr).Namespace("default").Get(context.TODO(), "cm2", v1.GetOptions{})
	require.NoError(t, err)

	data1, found, err := unstructured.NestedStringMap(obj1.Object, "data")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, map[string]string{"key1": "value1"}, data1)

	data2, found, err := unstructured.NestedStringMap(obj2.Object, "data")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, map[string]string{"key2": "value2"}, data2)
}

func TestDeleteMulti(t *testing.T) {
	kc := newFakeKubeClient(t)
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	// 预先创建两个 ConfigMap
	for _, name := range []string{"cm1", "cm2"} {
		cm := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name":      name,
					"namespace": "default",
				},
				"data": map[string]interface{}{
					"key": "value",
				},
			},
		}
		_, err := kc.dynamicClient.Resource(gvr).Namespace("default").Create(context.TODO(), cm, v1.CreateOptions{})
		require.NoError(t, err)
	}
	// 定义删除的 YAML
	yaml1 := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
`)
	yaml2 := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm2
`)
	yamls := [][]byte{yaml1, yaml2}
	err := kc.DeleteMulti(yamls, "default")
	require.NoError(t, err)

	// 验证删除
	_, err = kc.dynamicClient.Resource(gvr).Namespace("default").Get(context.TODO(), "cm1", v1.GetOptions{})
	assert.Error(t, err)
	_, err = kc.dynamicClient.Resource(gvr).Namespace("default").Get(context.TODO(), "cm2", v1.GetOptions{})
	assert.Error(t, err)
}

func TestApplyDir(t *testing.T) {
	kc := newFakeKubeClient(t)
	// 创建临时目录，并写入一个 YAML 文件
	tempDir, err := os.MkdirTemp("", "applydir")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "cm.yaml")
	yamlContent := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: dir-cm
data:
  dirkey: dirvalue
`)
	err = os.WriteFile(filePath, yamlContent, 0644)
	require.NoError(t, err)

	err = kc.ApplyDir(tempDir, "default")
	require.NoError(t, err)

	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	obj, err := kc.dynamicClient.Resource(gvr).Namespace("default").Get(context.TODO(), "dir-cm", v1.GetOptions{})
	require.NoError(t, err)
	data, found, err := unstructured.NestedStringMap(obj.Object, "data")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, map[string]string{"dirkey": "dirvalue"}, data)
}

func TestDeleteDir(t *testing.T) {
	kc := newFakeKubeClient(t)
	// 预先创建一个 ConfigMap
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	cm := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "dir-cm",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"dirkey": "dirvalue",
			},
		},
	}
	_, err := kc.dynamicClient.Resource(gvr).Namespace("default").Create(context.TODO(), cm, v1.CreateOptions{})
	require.NoError(t, err)

	// 创建临时目录，写入 YAML 用于删除
	tempDir, err := os.MkdirTemp("", "deletedir")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "cm.yaml")
	yamlContent := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: dir-cm
`)
	err = os.WriteFile(filePath, yamlContent, 0644)
	require.NoError(t, err)

	err = kc.DeleteDir(tempDir, "default")
	require.NoError(t, err)

	// 验证删除成功
	_, err = kc.dynamicClient.Resource(gvr).Namespace("default").Get(context.TODO(), "dir-cm", v1.GetOptions{})
	assert.Error(t, err)
}

func TestWithRealKubeClient(t *testing.T) {
	// 创建一个真实的 KubeClient 实例
	kc, err := NewKubeClient(util.Config{
		Namespace:        "open5gs",
		MonitorNamespace: "monarch",
		KubeconfigPath:   "/home/sming/.kube/config",
	})
	require.NoError(t, err)

	klog.InitFlags(nil)
	flag.Set("v", "6") // 日志级别调至最高

	// 验证能获取到 Pod 列表
	pods, err := kc.GetPods("open5gs")
	require.NoError(t, err)
	assert.NotEmpty(t, pods)
	// 打印pods为json
	fmt.Println(pods)
}
