package db

import (
	"slicer/model"
)

type Store interface {
	Querier
}

type Querier interface {
	// slice profile
	CreateSlice(slice *model.SliceProfile) error
	UpdateSlice(slice *model.SliceProfile) error
	DeleteSlice(id string) error
	GetSlice(id string) (*model.SliceProfile, error)
	GetSliceBySliceID(sliceID string) (*model.SliceProfile, error)
	ListSlice() ([]*model.SliceProfile, error)
	ListSliceID() ([]string, error)

	// monitor
	CreateMonitor(monitor *model.Monitor) error
	DeleteMonitor(id string) error
	GetMonitor(id string) (*model.Monitor, error)
	ListMonitor() ([]*model.Monitor, error)
}
