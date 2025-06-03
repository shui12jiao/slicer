package service

import (
	"slicer/db"
	"slicer/kube"
	"slicer/model"
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
		config.HelmReleasePrefix,
		config.CommonChartPath,
		config.SliceChartPath,
	)

	return &Service{
		Config:     config,
		Store:      store,
		KubeClient: kubeClient,
		HelmClient: helmClient,
		Open5gs:    open5gs,
	}
}

func (s *Service) GetOpen5gs() *Open5gs {
	// 返回Open5GS实例
	return s.Open5gs
}

func (s *Service) CreateSlice(slice model.SliceProfile) (model.SliceProfile, error) {
	// TODO
	return slice, nil
}

func (s *Service) UpdateSlice(slice model.SliceProfile) (model.SliceProfile, error) {
	// TODO
	return slice, nil
}

func (s *Service) DeleteSlice(sliceID string) error {
	// TODO
	return nil
}

func (s *Service) GetSlice(sliceID string) (*model.SliceProfile, error) {
	// TODO
	return nil, nil
}

func (s *Service) ListSlices() ([]model.SliceProfile, error) {
	// TODO
	return nil, nil
}
