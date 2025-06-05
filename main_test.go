package main

import (
	"slicer/api"
	"testing"
)

// 不需要依赖，进行server的简单测试
func TestSwagger(t *testing.T) {
	server := api.NewServer(api.NewSeverParam{})
	server.Start()
}
