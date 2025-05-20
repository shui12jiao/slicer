package kubeclient

import (
	"fmt"
	"log/slog"
	"slicer/util"

	"helm.sh/helm/v3/pkg/action"
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
	actionSet.install.Namespace = config.Namespace
	actionSet.upgrade.Namespace = config.Namespace
	actionSet.list.All = true // 默认列出所有 release

	return &HelmClient{
		config:  config,
		hconfig: actionConfig,
		action:  actionSet,
	}, nil
}
