package server

import "net/http"

// for Monarch

// service orchestrator相关接口
func (s *Server) soGetSliceComponents(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func (s *Server) soCheckHealth(w http.ResponseWriter, r *http.Request) {
	// TODO
}

// nfv orchestration相关接口
func (s *Server) noMdeInstall(w http.ResponseWriter, r *http.Request) {
	// TODO
}
func (s *Server) noMdeUninstall(w http.ResponseWriter, r *http.Request) {
	// TODO
}
func (s *Server) noMdeCheck(w http.ResponseWriter, r *http.Request) {
	// TODO
}
func (s *Server) noKpiComputationInstall(w http.ResponseWriter, r *http.Request) {
	// TODO
}
func (s *Server) noKpiComputationUninstall(w http.ResponseWriter, r *http.Request) {
	// TODO
}
func (s *Server) noKpiComputationCheck(w http.ResponseWriter, r *http.Request) {
	// TODO
}
func (s *Server) noCheckHealth(w http.ResponseWriter, r *http.Request) {
	// TODO
}
