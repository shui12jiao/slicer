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

	_ "slicer/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// Server 负责处理HTTP请求
type Server struct {
	router     *http.ServeMux
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
		router:     http.NewServeMux(),
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
	// swagger相关路由
	s.router.HandleFunc("GET /swagger/{any}", httpSwagger.WrapHandler)
	// for test
	s.router.HandleFunc("/ok",
		func(w http.ResponseWriter, r *http.Request) {
			slog.Info("ok")
			w.Write([]byte("ok"))
		})
	s.router.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		slog.Error("panic")
		panic("panic")
	})

	// 切片管理相关路由
	s.router.HandleFunc("POST /slice", s.createSlice)
	s.router.HandleFunc("DELETE /slice/{sliceId}", s.deleteSlice)
	s.router.HandleFunc("GET /slice/{sliceId}", s.getSlice)
	s.router.HandleFunc("GET /slice", s.listSlice)

	// 监控相关路由(目前只支持切片监控)
	s.router.HandleFunc("POST /monitor", s.createMonitor)
	s.router.HandleFunc("POST /monitor/external", s.createMonitorExternal) // 基于Monarch外部服务创建监控
	s.router.HandleFunc("DELETE /monitor/{monitorId}", s.deleteMonitor)
	s.router.HandleFunc("DELETE /monitor/external/{monitorId}", s.deleteMonitorExternal) // 基于Monarch外部服务删除监控
	s.router.HandleFunc("GET /monitor/{monitorId}", s.getMonitor)
	s.router.HandleFunc("GET /monitor", s.listMonitor)
	s.router.HandleFunc("GET /monitor/supported_kpis", s.getSupportedKpis)

	// 性能控制相关路由
	s.router.HandleFunc("POST /play", s.createPlay) // 或许应该合并到切片管理相关路由逻辑中, 生成默认值? 不允许删除
	s.router.HandleFunc("POST /play/{playId}", s.updatePlay)
	s.router.HandleFunc("GET /play/{playId}", s.getPlay)

	// SLA相关路由
	s.router.HandleFunc("POST /sla", s.createSla) //这里同时会将slice添加到controller中
	s.router.HandleFunc("POST /sla/{slaId}", s.updateSla)
	s.router.HandleFunc("DELETE /sla/{slaId}", s.deleteSla)
	s.router.HandleFunc("GET /sla/{slaId}", s.getSla)
	s.router.HandleFunc("GET /sla", s.listSla) // 列出所有SLA, 也用于controller获取 (controller没有持久化, 可能丢失)

	// Controller相关路由
	s.router.HandleFunc("GET /controller", s.getController)     // 获取controller的状态 包括切片列表, 策略等
	s.router.HandleFunc("POST /controller", s.updateController) // 更新controller的状态

	// Monarch交互相关路由
	// monarch调用service orchestrator相关接口
	s.router.HandleFunc("GET /service-orchestrator/slices/{sliceId}", s.soGetSliceComponents)
	s.router.HandleFunc("GET /service-orchestrator/api/health", s.soCheckHealth)
	// monarch调用nfv orchestration相关接口
	s.router.HandleFunc("POST /nfv-orchestrator/mde/install", s.noMdeInstall)
	s.router.HandleFunc("POST /nfv-orchestrator/mde/uninstall", s.noMdeUninstall)
	s.router.HandleFunc("POST /nfv-orchestrator/mde/check", s.noMdeCheck)
	s.router.HandleFunc("POST /nfv-orchestrator/kpi-computation/install", s.noKpiComputationInstall)
	s.router.HandleFunc("POST /nfv-orchestrator/kpi-computation/uninstall", s.noKpiComputationUninstall)
	s.router.HandleFunc("POST /nfv-orchestrator/kpi-computation/check", s.noKpiComputationCheck)
	s.router.HandleFunc("GET /nfv-orchestrator/api/health", s.noCheckHealth)
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
