package main

import (
	"log"
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
	if err != nil {
		log.Fatalf("初始化监控系统失败: %v", err)
	}

	// 连接数据库
	store, err := db.NewMongoDB(config)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 初始化渲染器
	render := render.NewRender(config)

	// 初始化Kubernetes客户端
	kubeclient, err := kubeclient.NewKubeClient(config)
	if err != nil {
		log.Fatalf("创建Kubernetes客户端失败: %v", err)
	}

	// 初始化IPAM
	ipam, err := db.NewIPAM(config)
	if err != nil {
		log.Fatalf("创建IP地址管理失败: %v", err)
	}

	// 初始化Server
	server := server.NewServer(config, monitor, store, ipam, render, kubeclient)

	server.Start()
}
