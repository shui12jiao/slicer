package main

import (
	"log/slog"
	"os"
	"slicer/ai"
	"slicer/api"
	"slicer/controller"
	"slicer/db"
	"slicer/kube"
	"slicer/monitor"
	"slicer/service"
	"slicer/util"
	"time"

	"github.com/lmittmann/tint"
)

// swag注释描述server信息
// @title Slicer API
// @version 1.0
// @description Slicer API
// @description 基于Kubernetes资源的切片管理系统API
// @description 包括 切片管理 监控管理 性能保证 等功能
// @host localhost:30001
// @BasePath /
func main() {
	// 采用slog作为日志库
	// slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	// 	Level: slog.LevelDebug,
	// })))
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug, // 设置日志级别
		TimeFormat: time.DateTime,   // 设置时间格式，例如 "3:04PM"
	})))

	// 加载配置
	config := util.LoadConfig(".env")

	// 连接数据库
	slog.Debug("连接数据库", "address", config.MongoURI, "database", config.MongoDBName)
	store, err := db.NewMongoDB(config)
	if err != nil {
		slog.Error("连接数据库失败", "error", err)
		os.Exit(1)
	}

	// 初始化Kubernetes客户端
	kubeClient, err := kube.NewKubeClient(config)
	if err != nil {
		slog.Error("创建Kubernetes客户端失败", "error", err)
		os.Exit(1)
	}

	// 初始化helm客户端
	helmClient, err := kube.NewHelmClient(config, kubeClient.GetKubeConfig())
	if err != nil {
		slog.Error("创建Helm客户端失败", "error", err)
		os.Exit(1)
	}

	// 初始化IPAM
	ipam, err := db.NewIPAM(config)
	if err != nil {
		slog.Error("初始化IPAM失败", "error", err)
		os.Exit(1)
	}

	// 初始化monitor监控系统交互组件
	monitor := monitor.NewMonitor(config)
	if err := monitor.Init(kubeClient); err != nil {
		slog.Error("初始化监控系统失败", "error", err)
		os.Exit(1)
	}

	// 初始化ai服务
	ai, err := ai.NewGeneralAI(config)
	if err != nil {
		slog.Error("创建AI失败", "error", err)
		os.Exit(1)
	}

	// 启动控制器
	controller := runController(config, store, ai, kubeClient)

	// 初始化Server
	server := api.NewServer(api.NewSeverParam{
		Service: service.NewService(
			service.NewServiceParam{
				Config:     config,
				Store:      store,
				IPAM:       ipam,
				KubeClient: kubeClient,
				HelmClient: helmClient,
				AI:         ai, // 可选的AI服务
			},
		),
		HelmClient: helmClient,
		Monitor:    monitor,
		Controller: controller,
	})

	// 启动HTTP服务器
	slog.Info("启动HTTP服务器", "address", config.HTTPServerAddress)
	server.Start()
}

// 注册并启动controller
func runController(config *util.Config, store db.Store, aiStrategy ai.AI, kclient *kube.KubeClient) controller.Controller {
	basicStrategy := newBasicStrategy(config)
	controller := controller.NewBasicController(config, store, kclient, aiStrategy, basicStrategy)
	controller.Start()
	slog.Info("控制器已启动", "频率", controller.GetFrequency(), "策略", controller.GetStrategy().Name())
	return controller
}

// 测试用基本策略
func newBasicStrategy(config *util.Config) controller.Strategy {
	// 初始化metrics源
	metrics, err := controller.NewMetrics(config.MonarchThanosURI)
	if err != nil {
		slog.Error("创建指标源失败", "error", err)
		os.Exit(1)
	}
	// 初始化策略
	return controller.NewBasicStrategy(metrics)
}
