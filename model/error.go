package model

import "errors"

var (
	ErrSliceAlreadyExists = errors.New("slice already exists")
	ErrSliceNotFound      = errors.New("slice not found")
)
