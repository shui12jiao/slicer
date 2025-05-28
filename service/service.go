package service

import (
	"log/slog"
	"slicer/db"
	"slicer/kube"
	"slicer/kube/value"
	"slicer/util"
)

type Service struct {
	Config     *util.Config
	Store      db.Store
	KubeClient *kube.KubeClient
	HelmClient *kube.HelmClient
	Open5gs    *Open5gs
}

func NewService(config *util.Config, store db.Store, kubeClient *kube.KubeClient, helmClient *kube.HelmClient) *Service {
	open5gs := NewOpen5gs(
		config.Namespace,
		config.HelmChartPath,
		config.HelmReleaseName,
		value.Open5gs{},
	)

	return &Service{
		Config:     config,
		Store:      store,
		KubeClient: kubeClient,
		HelmClient: helmClient,
		Open5gs:    open5gs,
	}
}

func (s *Service) ApplyOpen5gs(vals value.Open5gs) error {
	release, err := s.HelmClient.InstallOrUpgrade(s.Config.HelmReleaseName, s.Config.HelmChartPath, vals.ToMap())
	if err != nil {
		slog.Error("安装或升级 Open5gs 失败", "error", err)
		return err
	} else {
		slog.Info("Open5gs 安装或升级成功", "release", release.Name, "version", release.Version)
		// 更新 Open5gs 的 HelmValues
		s.Open5gs.HelmValues = vals
		s.Open5gs.Version = release.Version
		// 存储 Open5gs 的状态到数据库
		// if _, err := s.Store.CreateOpen5gs(s.Open5gs); err != nil {
		// 	slog.Error("保存 Open5gs 状态到数据库失败", "error", err)
		// }

		return nil
	}
}
