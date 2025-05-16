package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slicer/controller"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/monitor"
	"slicer/render"
	"slicer/util"

	"github.com/go-chi/chi"

	_ "slicer/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// Server 负责处理HTTP请求
type Server struct {
	router     *chi.Mux
	config     util.Config
	monitor    *monitor.Monitor
	store      db.Store
	ipam       *db.IPAM
	render     *render.Render
	kubeclient *kubeclient.KubeClient
	controller controller.Controller
}

type NewSeverArg struct {
	util.Config
	*monitor.Monitor
	db.Store
	*db.IPAM
	*render.Render
	*kubeclient.KubeClient
	controller.Controller
}

func NewServer(arg NewSeverArg) *Server {
	s := &Server{
		// router:     http.NewServeMux(),
		router:     chi.NewRouter(),
		config:     arg.Config,
		monitor:    arg.Monitor,
		store:      arg.Store,
		ipam:       arg.IPAM,
		render:     arg.Render,
		kubeclient: arg.KubeClient,
		controller: arg.Controller,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// 添加中间件
	s.router.Use(
		SlogLogger(slog.Default()),
		CORS(),
	)

	// swagger相关路由
	s.router.Get("/swagger/*", httpSwagger.WrapHandler)
	// for test
	s.router.Get("/ok", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("ok")
		w.Write([]byte("ok"))
	})
	s.router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		slog.Error("panic")
		panic("panic")
	})

	// 切片管理相关路由
	s.router.Post("/slice", s.createSlice)
	s.router.Delete("/slice/{sliceId}", s.deleteSlice)
	s.router.Get("/slice/{sliceId}", s.getSlice)
	s.router.Get("/slice", s.listSlice)

	// 监控相关路由(目前只支持切片监控)
	s.router.Post("/monitor", s.createMonitor)
	s.router.Post("/monitor/external", s.createMonitorExternal) // 基于Monarch外部服务创建监控
	s.router.Delete("/monitor/{monitorId}", s.deleteMonitor)
	s.router.Delete("/monitor/external/{monitorId}", s.deleteMonitorExternal) // 基于Monarch外部服务删除监控
	s.router.Get("/monitor/{monitorId}", s.getMonitor)
	s.router.Get("/monitor", s.listMonitor)
	s.router.Get("/monitor/supported_kpis", s.getSupportedKpis)

	// 性能控制相关路由
	s.router.Post("/play", s.createPlay) // 或许应该合并到切片管理相关路由逻辑中, 生成默认值? 不允许删除
	s.router.Post("/play/{playId}", s.updatePlay)
	s.router.Get("/play/{playId}", s.getPlay)

	// SLA相关路由
	s.router.Post("/sla", s.createSla) //这里同时会将slice添加到controller中
	s.router.Post("/sla/{slaId}", s.updateSla)
	s.router.Delete("/sla/{slaId}", s.deleteSla)
	s.router.Get("/sla/{slaId}", s.getSla)
	s.router.Get("/sla", s.listSla) // 列出所有SLA, 也用于controller获取 (controller没有持久化, 可能丢失)

	// Controller相关路由
	s.router.Get("/controller", s.getController)     // 获取controller的状态 包括切片列表, 策略等
	s.router.Post("/controller", s.updateController) // 更新controller的状态

	// Monarch交互相关路由
	// monarch调用service orchestrator相关接口
	s.router.Get("/service-orchestrator/slices/{sliceId}", s.soGetSliceComponents)
	s.router.Get("/service-orchestrator/api/health", s.soCheckHealth)
	// monarch调用nfv orchestration相关接口
	s.router.Post("/nfv-orchestrator/mde/install", s.noMdeInstall)
	s.router.Post("/nfv-orchestrator/mde/uninstall", s.noMdeUninstall)
	s.router.Post("/nfv-orchestrator/mde/check", s.noMdeCheck)
	s.router.Post("/nfv-orchestrator/kpi-computation/install", s.noKpiComputationInstall)
	s.router.Post("/nfv-orchestrator/kpi-computation/uninstall", s.noKpiComputationUninstall)
	s.router.Post("/nfv-orchestrator/kpi-computation/check", s.noKpiComputationCheck)
	s.router.Get("/nfv-orchestrator/api/health", s.noCheckHealth)
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	return http.ListenAndServe(s.config.HTTPServerAddress, s.router)
}

func encodeResponse(w http.ResponseWriter, response any) {
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "编码失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
