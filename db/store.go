package db

import "slicer/model"

type Store interface {
	Querier
}

type Querier interface {
	CreateSlice(slice model.SliceAndAddress) (model.SliceAndAddress, error)
	DeleteSlice(id string) error
	GetSlice(id string) (model.SliceAndAddress, error)
	GetSliceBySliceID(sliceID string) (model.SliceAndAddress, error)
	ListSlice() ([]model.SliceAndAddress, error)
	ListSliceID() ([]string, error)

	CreateMonitor(monitor model.Monitor) (model.Monitor, error)
	DeleteMonitor(id string) error
	GetMonitor(id string) (model.Monitor, error)
	ListMonitor() ([]model.Monitor, error)

	// CreateMonitor(monitor model.Monitor) (model.Monitor, error)
	// DeleteMonitor(id string) error
	// GetMonitor(id string) (model.Monitor, error)
	// ListMonitor() ([]model.Monitor, error)
}
