package kube

import (
	"errors"
	"fmt"
	"log/slog"
	"slicer/util"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
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
	get       *action.Get
	list      *action.List
	rollback  *action.Rollback
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
		slog.Error("Helm 配置初始化失败", "error", err)
		return nil, fmt.Errorf("Helm 配置初始化失败: %w", err)
	}

	// actionSet，封装常用的四大操作
	actionSet := &actionSet{
		install:   action.NewInstall(actionConfig),
		upgrade:   action.NewUpgrade(actionConfig),
		uninstall: action.NewUninstall(actionConfig),
		get:       action.NewGet(actionConfig),
		list:      action.NewList(actionConfig),
		rollback:  action.NewRollback(actionConfig),
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
	actionSet.upgrade.Namespace = config.Namespace
	// 设置卸载参数
	actionSet.uninstall.Wait = true
	actionSet.uninstall.Timeout = config.HelmTimeout
	// 设置列表参数
	actionSet.list.All = true            // 默认列出所有 release
	actionSet.list.AllNamespaces = false // 仅当前命名空间
	// 设置回滚参数
	actionSet.rollback.Wait = true
	actionSet.rollback.Timeout = config.HelmTimeout
	actionSet.rollback.DisableHooks = true // 禁用钩子，避免回滚时触发不必要的操作

	return &HelmClient{
		config:  config,
		hconfig: actionConfig,
		action:  actionSet,
	}, nil
}

// InstallOrUpgrade 安装或升级 Chart
func (hc *HelmClient) InstallOrUpgrade(releaseName, chartPath string, values map[string]interface{}) (*release.Release, error) {
	hc.action.upgrade.Install = true // 安装或升级 !并不会在不存在时自动安装!

	// 检查是否已存在
	exists, err := hc.ChartExists(releaseName)
	if err != nil {
		slog.Error("检测 Chart 是否存在时发生错误", "发布名称", releaseName, "error", err)
		return nil, fmt.Errorf("检测 Chart 是否存在时发生错误: %w", err)
	}
	if exists {
		// 如果存在，则执行升级
		slog.Info("检测到 Chart 已存在，开始执行升级操作", "发布名称", releaseName)
		return hc.Upgrade(releaseName, chartPath, values)
	} else {
		// 如果不存在，则执行安装
		slog.Info("未检测到 Chart，开始执行安装操作", "发布名称", releaseName)
		return hc.Install(releaseName, chartPath, values)
	}
}

func (hc *HelmClient) ChartExists(releaseName string) (bool, error) {
	if releaseName == "" {
		return false, fmt.Errorf("release 名称不能为空")
	}

	_, err := hc.action.get.Run(releaseName)
	if err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			// release 不存在
			return false, nil
		}
		// 其他错误
		slog.Error("查询 release 时发生内部错误", "发布名称", releaseName, "error", err)
		return false, fmt.Errorf("内部错误: %w", err)
	}

	// 查询成功，说明存在
	return true, nil
}

// Install 安装 Chart
func (hc *HelmClient) Install(releaseName, chartPath string, values map[string]interface{}) (*release.Release, error) {
	hc.action.install.ReleaseName = releaseName

	// 加载 Chart
	chartReq, err := loader.Load(chartPath)
	if err != nil {
		slog.Error("加载 Chart 失败", "Chart 路径", chartPath, "error", err)
		return nil, fmt.Errorf("加载 Chart 失败: %w", err)
	}

	// 执行安装
	rel, err := hc.action.install.Run(chartReq, values)
	if err != nil {
		slog.Error("部署 Chart 失败", "发布名称", releaseName, "error", err)
		return nil, fmt.Errorf("部署 Chart 失败: %w", err)
	}
	return rel, nil
}

// Upgrade 升级 Chart
// 注意：此方法仅在 Chart 已存在时使用
func (hc *HelmClient) Upgrade(releaseName, chartPath string, values map[string]interface{}) (*release.Release, error) {
	hc.action.upgrade.Install = false // 仅升级

	// 加载 Chart
	chartReq, err := loader.Load(chartPath)
	if err != nil {
		slog.Error("加载 Chart 失败", "Chart 路径", chartPath, "error", err)
		return nil, fmt.Errorf("加载 Chart 失败: %w", err)
	}

	// 执行升级
	rel, err := hc.action.upgrade.Run(releaseName, chartReq, values)
	if err != nil {
		slog.Error("升级 Chart 失败", "发布名称", releaseName, "error", err)
		return nil, fmt.Errorf("升级 Chart 失败: %w", err)
	}
	return rel, nil
}

// Uninstall 卸载 Chart
func (hc *HelmClient) Uninstall(releaseName string) error {
	// 执行卸载
	if _, err := hc.action.uninstall.Run(releaseName); err != nil {
		slog.Error("卸载 Chart 失败", "发布名称", releaseName, "error", err)
		return fmt.Errorf("卸载 Chart 失败: %w", err)
	}
	return nil
}

// Get 获取 Release 信息
func (hc *HelmClient) Get(releaseName string) (*release.Release, error) {
	if releaseName == "" {
		return nil, fmt.Errorf("release 名称不能为空")
	}

	// 执行查询
	rel, err := hc.action.get.Run(releaseName)
	if err != nil {
		slog.Error("获取 Release 信息失败", "发布名称", releaseName, "error", err)
		return nil, fmt.Errorf("获取 Release 信息失败: %w", err)
	}
	slog.Info("获取 Release 信息成功", "发布名称", releaseName, "版本", rel.Version)
	return rel, nil
}

// List 列出 Release
func (hc *HelmClient) List() ([]*release.Release, error) {
	// 执行查询
	release, err := hc.action.list.Run()
	if err != nil {
		slog.Error("获取 Release 列表失败", "error", err)
		return nil, fmt.Errorf("获取 Release 列表失败: %w", err)
	}

	return release, nil
}

// Rollback 回滚 Release
func (hc *HelmClient) Rollback(releaseName string, revision int) error {
	hc.action.rollback.Version = revision

	if revision < 0 { // 回滚版本号小于0,无效
		slog.Error("回滚版本号无效", "发布名称", releaseName)
		return fmt.Errorf("回滚版本号无效: %d", revision)
	} else if revision == 0 { //回滚到上一个版本
		slog.Info("回滚到上一个版本", "发布名称", releaseName)
	} else {
		slog.Info("回滚到指定版本", "发布名称", releaseName, "修订版本", revision)
	}

	// 执行回滚
	if err := hc.action.rollback.Run(releaseName); err != nil {
		slog.Error("回滚 Release 失败", "发布名称", releaseName, "修订版本", revision, "error", err)
		return fmt.Errorf("回滚 Release 失败: %w", err)
	}
	return nil
}
