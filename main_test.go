package main

import (
	"slicer/server"
	"slicer/util"
	"testing"
)

// 不需要依赖，进行server的简单测试
func TestSwagger(t *testing.T) {
	server := server.NewServer(server.NewSeverArg{
		Config: util.Config{
			ServerConfig: util.ServerConfig{
				HTTPServerAddress: "localhost:30001",
			},
		},
		Monitor:    nil,
		Store:      nil,
		IPAM:       nil,
		Render:     nil,
		KubeClient: nil,
		Controller: nil,
	})
	server.Start()
}
