package service

import (
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
