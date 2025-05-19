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

	// swagger
	s.router.Get("/swagger/*", httpSwagger.WrapHandler)

	// 简单测试
	s.router.Route("/", func(r chi.Router) {
		r.Get("/ok", func(w http.ResponseWriter, r *http.Request) {
			// 简单健康检查
			slog.Info("ok")
			w.Write([]byte("ok"))
		})
		r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
			// 模拟 panic，用于测试 Recoverer 中间件
			slog.Error("panic")
			panic("panic")
		})
	})

	// 切片管理
	s.router.Route("/slice", func(r chi.Router) {
		r.Post("/", s.createSlice)             // 创建新切片
		r.Delete("/{slice_id}", s.deleteSlice) // 删除指定切片
		r.Get("/{slice_id}", s.getSlice)       // 获取指定切片详情
		r.Get("/", s.listSlice)                // 列出所有切片
	})

	// 监控管理(目前仅支持切片监控)
	s.router.Route("/monitor", func(r chi.Router) {
		r.Post("/", s.createMonitor)                                // 创建切片监控
		r.Post("/external", s.createMonitorExternal)                // 基于Monarch外部服务创建监控
		r.Delete("/{monitor_id}", s.deleteMonitor)                  // 删除切片监控
		r.Delete("/external/{monitor_id}", s.deleteMonitorExternal) // 基于Monarch外部服务删除监控
		r.Get("/{monitor_id}", s.getMonitor)                        // 获取监控详情
		r.Get("/", s.listMonitor)                                   // 列出所有监控
		r.Get("/supported_kpis", s.getSupportedKpis)                // 获取支持的 KPI 列表
	})

	// 性能控制
	s.router.Route("/play", func(r chi.Router) {
		r.Post("/", s.createPlay)          // 创建性能控制任务（Play）
		r.Post("/{play_id}", s.updatePlay) // 更新指定 Play
		r.Get("/{play_id}", s.getPlay)     // 获取指定 Play 详情
	})

	// SLA管理
	s.router.Route("/sla", func(r chi.Router) {
		r.Post("/", s.createSla)           // 创建 SLA（此操作同时会将 slice 添加到 controller 中）
		r.Post("/{sla_id}", s.updateSla)   // 更新指定 SLA
		r.Delete("/{sla_id}", s.deleteSla) // 删除指定 SLA
		r.Get("/{sla_id}", s.getSla)       // 获取指定 SLA 详情
		r.Get("/", s.listSla)              // 列出所有 SLA（也用于 controller 获取）
	})

	// Controller管理
	s.router.Route("/controller", func(r chi.Router) {
		r.Get("/", s.getController)     // 获取 controller 的状态，包括切片列表、策略等
		r.Post("/", s.updateController) // 更新 controller 的状态
	})

	// Monarch交互
	// Monarch 调用 Service Orchestrator
	s.router.Route("/service-orchestrator", func(r chi.Router) {
		r.Get("/slices/{slice_id}", s.soGetSliceComponents) // 获取切片组件信息
		r.Get("/api/health", s.soCheckHealth)               // Service Orchestrator 健康检查
	})
	// Monarch 调用 NFV Orchestrator
	s.router.Route("/nfv-orchestrator", func(r chi.Router) {
		r.Route("/mde", func(r chi.Router) {
			r.Post("/install", s.noMdeInstall)     // 安装 MDE
			r.Post("/uninstall", s.noMdeUninstall) // 卸载 MDE
			r.Post("/check", s.noMdeCheck)         // 检查 MDE
		})
		r.Route("/kpi-computation", func(r chi.Router) {
			r.Post("/install", s.noKpiComputationInstall)     // 安装 KPI 计算服务
			r.Post("/uninstall", s.noKpiComputationUninstall) // 卸载 KPI 计算服务
			r.Post("/check", s.noKpiComputationCheck)         // 检查 KPI 计算服务
		})
		r.Get("/api/health", s.noCheckHealth) // NFV Orchestrator 健康检查
	})
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
