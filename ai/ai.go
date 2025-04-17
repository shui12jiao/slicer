package ai

import "slicer/controller"

type AI interface {
	// Strategy实现
	controller.Strategy
}
