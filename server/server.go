package server

import (
	"net/http"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/render"
	"slicer/util"
)

// Server 负责处理HTTP请求
type Server struct {
	router     *http.ServeMux
	config     util.Config
	store      db.Store
	ipam       *db.IPAM
	render     *render.Render
	kubeclient *kubeclient.KubeClient
}

func NewServer(config util.Config, store db.Store, ipam *db.IPAM, render *render.Render, kubeclient *kubeclient.KubeClient) *Server {
	s := &Server{
		router:     http.NewServeMux(),
		config:     config,
		store:      store,
		ipam:       ipam,
		render:     render,
		kubeclient: kubeclient,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.HandleFunc("POST /slice", s.createSlice)
	s.router.HandleFunc("DELETE /slice/{sliceId}", s.deleteSlice)
	s.router.HandleFunc("GET /slice/{sliceId}", s.getSlice)
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	return http.ListenAndServe(s.config.HTTPServerAddress, s.router)
}
