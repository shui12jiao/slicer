package main

import (
	"fmt"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/monitor"
	"slicer/render"
	"slicer/server"
	"slicer/util"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	config := util.LoadConfig()

	// 初始化monitor监控系统交互组件
	monitor, err := monitor.NewMonitor(config)

	// 连接数据库
	store, err := db.NewMongoDB(config)
	if err != nil {
		panic(fmt.Sprintf("连接数据库失败: %v", err))
	}

	// 初始化渲染器
	render := render.NewRender(config)

	// 初始化Kubernetes客户端
	kubeclient, err := kubeclient.NewKubeClient(config.KubeconfigPath)
	if err != nil {
		panic(fmt.Sprintf("创建Kubernetes客户端失败: %v", err))
	}

	// 初始化IPAM
	ipam, err := db.NewIPAM(config)
	if err != nil {
		panic(fmt.Sprintf("创建IP地址管理失败: %v", err))
	}

	// 初始化Server
	server := server.NewServer(config, monitor, store, ipam, render, kubeclient)

	server.Start()
}
