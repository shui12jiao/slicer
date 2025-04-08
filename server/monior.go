package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slicer/model"
)

// 获取支持的KPI
// GET /monitor/supported_kpis
type getSupportedKpisResponse struct {
	SupportedKpis []model.SupportedKpi `json:"supported_kpis"`
}

func (s *Server) getSupportedKpis(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取支持的KPI请求", "method", r.Method, "url", r.URL.String())

	kpis, err := s.monitor.GetSupportedKpis()
	if err != nil {
		slog.Error("获取支持的KPI失败", "error", err)
		http.Error(w, "获取支持的KPI失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取支持的KPI成功", "count", len(kpis))
	encodeResponse(w, getSupportedKpisResponse{SupportedKpis: kpis})
}

// 创建监控请求
// POST /monitor
type createMonitorRequest struct {
	Monitor model.Monitor `json:"monitor"`
}

type createMonitorResponse = createMonitorRequest

func (s *Server) createMonitor(w http.ResponseWriter, r *http.Request) {
	slog.Debug("创建监控请求", "method", r.Method, "url", r.URL.String())

	// 解析请求
	var createMonitorRequest createMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&createMonitorRequest); err != nil {
		slog.Warn("请求解码失败", "error", err)
		http.Error(w, "请求解码失败: "+err.Error(), http.StatusBadRequest)
		return
	}
	monitor := createMonitorRequest.Monitor

	// 检查请求参数
	if err := monitor.Validate(); err != nil {
		slog.Warn("请求验证失败", "error", err)
		http.Error(w, "请求验证失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 获取sliceId
	sliceId := monitor.KPI.SubCounter.SubCounterIDs[0]
	if sliceId == "" {
		slog.Warn("缺少sliceId参数")
		http.Error(w, "缺少sliceId参数", http.StatusBadRequest)
		return
	}

	// 发送监控请求
	monitor, err := s.monitor.SubmitMonitoring(monitor)
	if err != nil {
		slog.Error("提交监控请求失败", "sliceID", sliceId, "error", err)
		http.Error(w, "提交监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 存储监控请求
	monitor, err = s.store.CreateMonitor(monitor)
	if err != nil {
		slog.Error("存储监控请求失败", "sliceID", sliceId, "error", err)
		http.Error(w, "存储监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("创建监控请求成功", "sliceID", sliceId, "monitorID", monitor.ID.Hex())
	encodeResponse(w, createMonitorResponse{Monitor: monitor})
}

// 删除监控请求
// DELETE /monitor/slice/{monitorId}
func (s *Server) deleteMonitor(w http.ResponseWriter, r *http.Request) {
	slog.Debug("删除监控请求", "method", r.Method, "url", r.URL.String())

	// 获取monitorId
	monitorId := r.PathValue("monitorId")
	if monitorId == "" {
		slog.Warn("缺少monitorId参数")
		http.Error(w, "缺少monitorId参数", http.StatusBadRequest)
		return
	}

	// 从monitor存储中获取sliceId
	monitor, err := s.store.GetMonitor(monitorId)
	if err != nil {
		slog.Error("获取监控请求失败", "monitorID", monitorId, "error", err)
		http.Error(w, "不存在该监控请求: "+err.Error(), http.StatusNotFound)
		return
	}

	// 获取sliceId
	sliceId := monitor.KPI.SubCounter.SubCounterIDs[0]

	// 删除MDE
	yaml, err := s.render.RenderMde(sliceId)
	if err != nil {
		slog.Error("渲染MDE yaml失败", "sliceID", sliceId, "error", err)
		http.Error(w, "渲染yaml失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.kubeclient.Delete(yaml, s.config.MonitorNamespace)

	// 删除KPI
	yaml, err = s.render.RenderKpiCalc(sliceId)
	if err != nil {
		slog.Error("渲染KPI yaml失败", "sliceID", sliceId, "error", err)
		http.Error(w, "渲染yaml失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.kubeclient.Delete(yaml, s.config.MonitorNamespace)

	// 存储中删除
	err = s.store.DeleteMonitor(monitorId)
	if err != nil {
		slog.Error("删除存储监控请求失败", "monitorID", monitorId, "error", err)
		http.Error(w, "删除存储监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("删除监控请求成功", "monitorID", monitorId, "sliceID", sliceId)
	w.WriteHeader(http.StatusOK)
}

// 获取监控请求
// GET /monitor/slice/{monitorId}
type getMonitorResponse struct {
	Monitor model.Monitor `json:"monitor"`
}

func (s *Server) getMonitor(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取监控请求", "method", r.Method, "url", r.URL.String())

	// 获取MonitorId
	monitorId := r.PathValue("monitorId")
	if monitorId == "" {
		slog.Warn("缺少monitorId参数")
		http.Error(w, "缺少monitorId参数", http.StatusBadRequest)
		return
	}

	// 获取Monitor
	monitor, err := s.store.GetMonitor(monitorId)
	if err != nil {
		slog.Error("获取监控请求失败", "monitorID", monitorId, "error", err)
		http.Error(w, "获取监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取监控请求成功", "monitorID", monitorId)
	encodeResponse(w, getMonitorResponse{Monitor: monitor})
}

// 获取监控请求列表
// GET /monitor
type listMonitorResponse struct {
	Monitors []model.Monitor `json:"monitors"`
}

func (s *Server) listMonitor(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取监控请求列表", "method", r.Method, "url", r.URL.String())

	monitors, err := s.store.ListMonitor()
	if err != nil {
		slog.Error("获取监控请求列表失败", "error", err)
		http.Error(w, "获取监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取监控请求列表成功", "count", len(monitors))
	encodeResponse(w, listMonitorResponse{Monitors: monitors})
}
