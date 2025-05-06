package main

import (
	"log/slog"
	"os"
	"slicer/ai"
	"slicer/controller"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/monitor"
	"slicer/render"
	"slicer/server"
	"slicer/util"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/lmittmann/tint"
)

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

	// 启动控制器
	controller := runController(config, store, kubeclient)

	// 初始化Server
	server := server.NewServer(server.NewSeverArg{
		Config:     config,
		Store:      store,
		KubeClient: kubeclient,
		Monitor:    monitor,
		Render:     render,
		IPAM:       ipam,
		Controller: controller,
	})

	// 启动HTTP服务器
	slog.Info("启动HTTP服务器", "address", config.HTTPServerAddress)
	server.Start()
}

// 注册并启动controller
func runController(config util.Config, store db.Store, kclient *kubeclient.KubeClient) controller.Controller {
	basicStrategy := newBasicStrategy(config)
	aiStrategy := newAIStrategy(config)
	controller := controller.NewBasicController(config, store, kclient, basicStrategy, aiStrategy)
	controller.Start()
	slog.Info("控制器已启动", "频率", controller.GetFrequency(), "策略", controller.GetStrategy().Name())
	return controller
}

// 测试用基本策略
func newBasicStrategy(config util.Config) controller.Strategy {
	// 初始化metrics源
	metrics, err := controller.NewMetrics(config.MonarchThanosURI)
	if err != nil {
		slog.Error("创建指标源失败", "error", err)
		os.Exit(1)
	}
	// 初始化策略
	return controller.NewBasicStrategy(metrics)
}

// ai大模型支持的策略
func newAIStrategy(config util.Config) controller.Strategy {
	// 初始化ai
	ai, err := ai.NewGeneralAI(config)
	if err != nil {
		slog.Error("创建AI失败", "error", err)
		os.Exit(1)
	}
	return ai
}
