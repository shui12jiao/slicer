package main

import (
	"log/slog"
	"os"
	"slicer/controller"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/monitor"
	"slicer/render"
	"slicer/server"
	"slicer/util"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// 采用slog作为日志库
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	// 加载配置
	config := util.LoadConfig()

	// 连接数据库
	slog.Debug("连接数据库", "address", config.MongoURI, "database", config.MongoDBName)
	store, err := db.NewMongoDB(config)
	if err != nil {
		slog.Error("连接数据库失败", "error", err)
		os.Exit(1)
	}

	// 初始化monitor监控系统交互组件
	monitor := monitor.NewMonitor(config)

	// 初始化渲染器
	render := render.NewRender(config)

	// 初始化Kubernetes客户端
	kubeclient, err := kubeclient.NewKubeClient(config)
	if err != nil {
		slog.Error("创建Kubernetes客户端失败", "error", err)
		os.Exit(1)
	}

	// 初始化IPAM
	ipam, err := db.NewIPAM(config)
	if err != nil {
		slog.Error("创建IP地址管理失败", "error", err)
		os.Exit(1)
	}

	// 初始化metrics源
	metrics, err := controller.NewMetrics(config.MonarchThanosURI)
	if err != nil {
		slog.Error("创建指标源失败", "error", err)
		os.Exit(1)
	}
	// 初始化策略
	strategy := controller.NewBasicStrategy(metrics)

	// 初始化控制器
	controller := controller.NewBasicController(config, store, kubeclient, strategy)

	// 启动控制器
	go controller.Run()

	// 初始化Server
	server := server.NewServer(config, monitor, store, ipam, render, kubeclient)

	// 启动HTTP服务器
	slog.Info("启动HTTP服务器", "address", config.HTTPServerAddress)
	server.Start()
}
