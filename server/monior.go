package server

import (
	"encoding/json"
	"net/http"
	"slicer/model"
)

// 获取支持的KPI
// GET /monitor/supported_kpis
type getSupportedKpisResponse struct {
	SupportedKpis []model.SupportedKpi `json:"supported_kpis"`
}

func (s *Server) getSupportedKpis(w http.ResponseWriter, r *http.Request) {
	kpis, err := s.monitor.GetSupportedKpis()
	if err != nil {
		http.Error(w, "获取支持的KPI失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	encodeResponse(w, getSupportedKpisResponse{SupportedKpis: kpis})
}

// 创建监控请求
// POST /monitor
type createMonitorRequest struct {
	Monitor model.Monitor `json:"monitor"`
}

type createMonitorResponse = createMonitorRequest

func (s *Server) createMonitor(w http.ResponseWriter, r *http.Request) {
	// 解析请求
	var createMonitorRequest createMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&createMonitorRequest); err != nil {
		http.Error(w, "请求解码失败: "+err.Error(), http.StatusBadRequest)
		return
	}
	monitor := createMonitorRequest.Monitor

	// 获取sliceId
	sliceId := monitor.SliceID
	if sliceId == "" {
		http.Error(w, "缺少sliceId参数", http.StatusBadRequest)
		return
	}

	// 验证请求
	if err := monitor.Validate(); err != nil {
		http.Error(w, "请求验证失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 发送监控请求
	monitor, err := s.monitor.SubmitMonitoring(monitor)
	if err != nil {
		http.Error(w, "提交监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 存储监控请求
	monitor, err = s.store.CreateMonitor(monitor)
	if err != nil {
		http.Error(w, "存储监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回响应
	encodeResponse(w, createMonitorResponse{Monitor: monitor})
}

// 删除监控请求
// DELETE /monitor/slice/{monitorId}
func (s *Server) deleteMonitor(w http.ResponseWriter, r *http.Request) {
	// 获取monitorId
	monitorId := r.PathValue("monitorId")
	if monitorId == "" {
		http.Error(w, "缺少monitorId参数", http.StatusBadRequest)
		return
	}

	// 发送删除监控请求
	// err := s.monitor.DeleteMonitoring(monitorId)
	// if err != nil {
	// 	http.Error(w, "提交监控请求失败: "+err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	// 从monitor存储中获取sliceId
	monitor, err := s.store.GetMonitor(monitorId)
	if err != nil {
		http.Error(w, "不存在该监控请求: "+err.Error(), http.StatusNotFound)
		return
	}

	// 这里不需要发送删除监控请求，直接进行删除
	// 删除MDE
	yaml, err := s.render.RenderMde([]string{monitor.SliceID})
	if err != nil {
		http.Error(w, "渲染yaml失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.kubeclient.Delete(yaml, s.config.MonitorNamespace)

	// 删除KPI
	yaml, err = s.render.RenderKpiComp([]string{monitor.SliceID})
	if err != nil {
		http.Error(w, "渲染yaml失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.kubeclient.Delete(yaml, s.config.MonitorNamespace)

	// 存储中删除
	err = s.store.DeleteMonitor(monitorId)
	if err != nil {
		http.Error(w, "删除存储监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回响应
	w.WriteHeader(http.StatusOK)
}

// 获取监控请求
// GET /monitor/slice/{monitorId}
type getMonitorResponse struct {
	Monitor model.Monitor `json:"monitor"`
}

func (s *Server) getMonitor(w http.ResponseWriter, r *http.Request) {
	// 获取MonitorId
	获取MonitorId := r.PathValue("获取MonitorId")
	if 获取MonitorId == "" {
		http.Error(w, "缺少获取MonitorId参数", http.StatusBadRequest)
		return
	}

	// 获取Monitor
	monitor, err := s.store.GetMonitor(获取MonitorId)
	if err != nil {
		http.Error(w, "获取监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	encodeResponse(w, getMonitorResponse{Monitor: monitor})
}

// 获取监控请求列表
// GET /monitor
type listMonitorResponse struct {
	Monitors []model.Monitor `json:"monitors"`
}

func (s *Server) listMonitor(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.store.ListMonitor()
	if err != nil {
		http.Error(w, "获取监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	encodeResponse(w, listMonitorResponse{Monitors: monitors})
}
