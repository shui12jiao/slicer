package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slicer/controller"
	"slicer/kube"
	"slicer/monitor"
	"slicer/service"
	"slicer/util"

	"github.com/go-chi/chi"

	_ "slicer/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// Server 负责处理HTTP请求
type Server struct {
	config     *util.Config // 配置
	router     *chi.Mux
	monitor    *monitor.Monitor
	controller controller.Controller
	service    *service.Service
}

type NewSeverArg struct {
	*service.Service
	*monitor.Monitor
	*kube.HelmClient
	controller.Controller
}

func NewServer(arg NewSeverArg) *Server {
	s := &Server{
		// router:     http.NewServeMux(),
		config:     arg.Service.Config,
		router:     chi.NewRouter(),
		monitor:    arg.Monitor,
		service:    arg.Service,
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
		r.Post("/", s.createSlice)             // 创建切片
		r.Delete("/{slice_id}", s.deleteSlice) // 删除切片
		r.Put("/{slice_id}", s.updateSlice)    // 更新整个切片
		r.Get("/{slice_id}", s.getSlice)       // 获取切片详情
		r.Get("/", s.listSlice)                // 获取全部切片列表

		// 更新 SLA
		r.Put("/{slice_id}/sla", s.updateSLA)

		// 更新 Deploy
		r.Put("/{slice_id}/deploy", s.updateDeploy)
	})

	// 监控管理(目前仅支持切片监控)
	s.router.Route("/monitor", func(r chi.Router) {
		r.Post("/", s.createMonitor)                 // 创建切片监控
		r.Delete("/{monitor_id}", s.deleteMonitor)   // 删除切片监控
		r.Get("/{monitor_id}", s.getMonitor)         // 获取监控详情
		r.Get("/", s.listMonitor)                    // 列出所有监控
		r.Get("/supported_kpis", s.getSupportedKpis) // 获取支持的 KPI 列表

		r.Route("/external", func(rr chi.Router) {
			rr.Post("/", s.createMonitorExternal)               // 基于Monarch外部服务创建监控
			rr.Delete("/{monitor_id}", s.deleteMonitorExternal) // 基于Monarch外部服务删除监控
		})
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
