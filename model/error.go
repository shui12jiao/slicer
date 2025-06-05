package model

import "errors"

var (
	ErrSliceAlreadyExists = errors.New("slice already exists")
	ErrSliceNotFound      = errors.New("slice not found in MongoDB")
	ErrMonitorNotFound    = errors.New("monitor not found in MongoDB")
)
