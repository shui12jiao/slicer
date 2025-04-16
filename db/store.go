package db

import "slicer/model"

type Store interface {
	Querier
}

type Querier interface {
	// slice
	CreateSlice(slice model.SliceAndAddress) (model.SliceAndAddress, error)
	DeleteSlice(id string) error
	GetSlice(id string) (model.SliceAndAddress, error)
	GetSliceBySliceID(sliceID string) (model.SliceAndAddress, error)
	ListSlice() ([]model.SliceAndAddress, error)
	ListSliceID() ([]string, error)

	// monitor
	CreateMonitor(monitor model.Monitor) (model.Monitor, error)
	DeleteMonitor(id string) error
	GetMonitor(id string) (model.Monitor, error)
	ListMonitor() ([]model.Monitor, error)

	// play
	CreatePlay(play model.Play) (model.Play, error)
	DeletePlay(id string) error
	GetPlay(id string) (model.Play, error)
	GetPlayBySliceID(sliceID string) (model.Play, error)
	ListPlay() ([]model.Play, error)
	UpdatePlay(play model.Play) (model.Play, error)

	// sla
	CreateSLA(sla model.SLA) (model.SLA, error)
	DeleteSLA(id string) error
	GetSLA(id string) (model.SLA, error)
	GetSLABySliceID(sliceID string) (model.SLA, error)
	ListSLA() ([]model.SLA, error)
	UpdateSLA(sla model.SLA) (model.SLA, error)
}
