package server

import (
	"net/http"
	"slicer/model"
)

type createMonitorRequest struct {
	Monitor model.Monitor `json:"monitor"`
}

func (s *Server) createMonitor(w http.ResponseWriter, r *http.Request) {
	// 获取sliceId
	sliceId := r.PathValue("sliceId")
	if sliceId == "" {
		http.Error(w, "缺少sliceId参数", http.StatusBadRequest)
		return
	}

}

func (s *Server) deleteMonitor(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) getMonitor(w http.ResponseWriter, r *http.Request) {

}
