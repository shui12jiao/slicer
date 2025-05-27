package kube

import (
	"fmt"
	"log/slog"
	"slicer/util"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

// HelmClient 定义helm客户端
type HelmClient struct {
	config  *util.Config          //应用配置
	hconfig *action.Configuration // Helm配置
	action  *actionSet            // 核心客户端集合
}

// 内部封装的action客户端集合
type actionSet struct {
	install   *action.Install
	upgrade   *action.Upgrade
	uninstall *action.Uninstall
	list      *action.List
}

func NewHelmClient(config *util.Config, kconfig *rest.Config) (*HelmClient, error) {
	// 不使用Helm的EnvSettings
	// 基于已有的rest.Config创建一个新的ConfigFlags
	// EnvSettings无法利用InClusterConfig，导致无法在集群内使用
	kubeConfig := genericclioptions.NewConfigFlags(false)
	kubeConfig.APIServer = &kconfig.Host
	kubeConfig.BearerToken = &kconfig.BearerToken
	kubeConfig.CAFile = &kconfig.CAFile
	kubeConfig.Namespace = &config.Namespace
	timeoutStr := kconfig.Timeout.String()
	kubeConfig.Timeout = &timeoutStr
	kubeConfig.Insecure = &kconfig.Insecure

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(
		kubeConfig,
		config.Namespace,
		config.HelmDriver,
		func(format string, v ...interface{}) {
			slog.Debug(fmt.Sprintf(format, v...))
		},
	); err != nil {
		slog.Error("初始化 Helm action configuration 失败", "error", err)
		return nil, fmt.Errorf("初始化 Helm action configuration 失败: %w", err)
	}

	// actionSet，封装常用的四大操作
	actionSet := &actionSet{
		install:   action.NewInstall(actionConfig),
		upgrade:   action.NewUpgrade(actionConfig),
		uninstall: action.NewUninstall(actionConfig),
		list:      action.NewList(actionConfig),
	}

	// 配置 Helm 操作的默认值
	// 设置安装参数
	actionSet.install.Namespace = config.Namespace
	actionSet.install.CreateNamespace = true
	actionSet.install.Wait = true
	actionSet.install.Timeout = config.HelmTimeout
	// 设置升级参数
	actionSet.upgrade.Wait = true
	actionSet.upgrade.Timeout = config.HelmTimeout
	actionSet.upgrade.Install = true // 允许不存在时自动安装
	actionSet.upgrade.Namespace = config.Namespace
	// 设置卸载参数
	actionSet.uninstall.Wait = true
	actionSet.uninstall.Timeout = time.Minute * 5
	// 设置列表参数
	actionSet.list.All = true            // 默认列出所有 release
	actionSet.list.AllNamespaces = false // 仅当前命名空间

	return &HelmClient{
		config:  config,
		hconfig: actionConfig,
		action:  actionSet,
	}, nil
}

// Install 安装 Chart
func (hc *HelmClient) Install(releaseName, chartPath string, values map[string]interface{}) (*release.Release, error) {
	hc.action.install.ReleaseName = releaseName

	// 加载 Chart
	chartReq, err := loader.Load(chartPath)
	if err != nil {
		slog.Error("加载chart失败", "path", chartPath, "error", err)
		return nil, fmt.Errorf("加载chart失败: %w", err)
	}

	// 执行安装
	rel, err := hc.action.install.Run(chartReq, values)
	if err != nil {
		slog.Error("部署chart失败", "release", releaseName, "error", err)
		return nil, fmt.Errorf("部署chart失败: %w", err)
	}
	return rel, nil
}

// Upgrade 升级 Chart
func (hc *HelmClient) Upgrade(releaseName, chartPath string, values map[string]interface{}) (*release.Release, error) {
	// 加载 Chart
	chartReq, err := loader.Load(chartPath)
	if err != nil {
		slog.Error("加载chart失败", "path", chartPath, "error", err)
		return nil, fmt.Errorf("加载chart失败: %w", err)
	}

	// 执行升级
	rel, err := hc.action.upgrade.Run(releaseName, chartReq, values)
	if err != nil {
		slog.Error("升级chart失败", "release", releaseName, "error", err)
		return nil, fmt.Errorf("升级chart失败: %w", err)
	}
	return rel, nil
}

// Uninstall 卸载 Chart
func (hc *HelmClient) Uninstall(releaseName string) error {
	// 执行卸载
	if _, err := hc.action.uninstall.Run(releaseName); err != nil {
		slog.Error("卸载chart失败", "release", releaseName, "error", err)
		return fmt.Errorf("卸载chart失败: %w", err)
	}
	return nil
}

// List 列出 Release
func (hc *HelmClient) List() ([]*release.Release, error) {
	// 执行查询
	results, err := hc.action.list.Run()
	if err != nil {
		slog.Error("获取release列表失败", "error", err)
		return nil, fmt.Errorf("获取release列表失败: %w", err)
	}

	// 过滤无效结果
	var validReleases []*release.Release
	for _, r := range results {
		if r != nil && r.Info != nil {
			validReleases = append(validReleases, r)
		}
	}
	return validReleases, nil
}
