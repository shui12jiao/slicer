package service

import (
	"slicer/db"
	"slicer/kube"
	"slicer/util"
)

type Service struct {
	Config     *util.Config
	Store      db.Store
	IPAM       *db.IPAM
	KubeClient *kube.KubeClient
	HelmClient *kube.HelmClient
	Open5gs    *Open5gs
}

type NewServiceParam struct {
	Config     *util.Config
	Store      db.Store
	IPAM       *db.IPAM
	KubeClient *kube.KubeClient
	HelmClient *kube.HelmClient
}

func NewService(param NewServiceParam) *Service {
	open5gs := NewOpen5gs(
		param.Config.Namespace,
		param.Config.HelmReleasePrefix,
		param.Config.CommonChartPath,
		param.Config.SliceChartPath,
	)

	return &Service{
		Config:     param.Config,
		Store:      param.Store,
		KubeClient: param.KubeClient,
		HelmClient: param.HelmClient,
		Open5gs:    open5gs,
	}
}
