package server

import (
	"encoding/json"
	"net/http"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/monitor"
	"slicer/render"
	"slicer/util"
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
}

func NewServer(config util.Config, monitor *monitor.Monitor, store db.Store, ipam *db.IPAM, render *render.Render, kubeclient *kubeclient.KubeClient) *Server {
	s := &Server{
		router:     http.NewServeMux(),
		config:     config,
		monitor:    monitor,
		store:      store,
		ipam:       ipam,
		render:     render,
		kubeclient: kubeclient,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// 切片管理相关路由
	s.router.HandleFunc("POST /slice", s.createSlice)
	s.router.HandleFunc("DELETE /slice/{sliceId}", s.deleteSlice)
	s.router.HandleFunc("GET /slice/{sliceId}", s.getSlice)
	s.router.HandleFunc("GET /slice", s.listSlice)

	// 监控相关路由(目前只支持切片监控)
	s.router.HandleFunc("POST /monitor/slice/{sliceId}", s.createMonitor)
	s.router.HandleFunc("DELETE /monitor/slice/{monitorId}", s.deleteMonitor)
	s.router.HandleFunc("GET /monitor/slice/{monitorId}", s.getMonitor)
	s.router.HandleFunc("GET /monitor", s.listMonitor)
	s.router.HandleFunc("GET /monitor/supported_kpis", s.getSupportedKpis)

	// Monarch交互相关路由
	// monarch调用service orchestrator相关接口
	s.router.HandleFunc("GET /service-orchestrator/slices/{sliceId}", s.soGetSliceComponents)
	s.router.HandleFunc("GET /service-orchestrator/api/health", s.soCheckHealth)
	// monarch调用nfv orchestration相关接口
	s.router.HandleFunc("POST /nfv-orchestrator/mde/install", s.noMdeInstall)
	// s.router.HandleFunc("POST /nfv-orchestrator/mde/uninstall", s.noMdeUninstall)
	s.router.HandleFunc("POST /nfv-orchestrator/mde/check", s.noMdeCheck)
	s.router.HandleFunc("POST /nfv-orchestrator/kpi-computation/install", s.noKpiComputationInstall)
	// s.router.HandleFunc("POST /nfv-orchestrator/kpi-computation/uninstall", s.noKpiComputationUninstall)
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
