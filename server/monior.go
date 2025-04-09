package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slicer/model"

	"go.mongodb.org/mongo-driver/mongo"
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

	// 渲染mde yaml
	yamlMde, err := s.render.RenderMde(sliceId)
	if err != nil {
		slog.Error("渲染MDE yaml失败", "sliceID", sliceId, "error", err)
		http.Error(w, "渲染yaml失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 渲染kpic yaml
	yamlKpi, err := s.render.RenderKpiCalc(sliceId)
	if err != nil {
		slog.Error("渲染KPI yaml失败", "sliceID", sliceId, "error", err)
		http.Error(w, "渲染yaml失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 部署MDE
	if err := s.kubeclient.Apply(yamlMde, s.config.MonitorNamespace); err != nil {
		slog.Error("部署MDE失败", "sliceID", sliceId, "error", err)
		http.Error(w, "部署MDE失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 部署KPI
	if err := s.kubeclient.Apply(yamlKpi, s.config.MonitorNamespace); err != nil {
		slog.Error("部署KPI失败", "sliceID", sliceId, "error", err)
		http.Error(w, "部署KPI失败: "+err.Error(), http.StatusInternalServerError)
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

// 创建监控请求(基于Monarch外部服务)
// createMonitorExternal

type createMonitorExternalRequest = createMonitorRequest

type createMonitorExternalResponse = createMonitorRequest

func (s *Server) createMonitorExternal(w http.ResponseWriter, r *http.Request) {
	slog.Debug("创建监控请求", "method", r.Method, "url", r.URL.String())

	// 解析请求
	var createMonitorRequest createMonitorExternalRequest
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
	encodeResponse(w, createMonitorExternalResponse{Monitor: monitor})
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
		if errors.Is(err, mongo.ErrNilDocument) { // MongoDB为空文档
			slog.Warn("监控不存在", "monitorID", monitorId)
			http.Error(w, fmt.Sprintf("monitor不存在: %v", monitorId), http.StatusNotFound)
			return
		}

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
	err = s.kubeclient.Delete(yaml, s.config.MonitorNamespace)
	if err != nil {
		slog.Error("删除MDE失败", "sliceID", sliceId, "error", err)
		http.Error(w, "删除MDE失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 删除KPI
	yaml, err = s.render.RenderKpiCalc(sliceId)
	if err != nil {
		slog.Error("渲染KPI yaml失败", "sliceID", sliceId, "error", err)
		http.Error(w, "渲染yaml失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = s.kubeclient.Delete(yaml, s.config.MonitorNamespace)
	if err != nil {
		slog.Error("删除KPI失败", "sliceID", sliceId, "error", err)
		http.Error(w, "删除KPI失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

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

// 删除监控请求(基于Monarch外部服务)
// DELETE /monitor/external/{monitorId}

func (s *Server) deleteMonitorExternal(w http.ResponseWriter, r *http.Request) {
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
		if errors.Is(err, mongo.ErrNilDocument) { // MongoDB为空文档
			slog.Warn("监控不存在", "monitorID", monitorId)
			http.Error(w, fmt.Sprintf("monitor不存在: %v", monitorId), http.StatusNotFound)
			return
		}

		slog.Error("获取监控请求失败", "monitorID", monitorId, "error", err)
		http.Error(w, "不存在该监控请求: "+err.Error(), http.StatusNotFound)
		return
	}

	// 获取sliceId
	sliceId := monitor.KPI.SubCounter.SubCounterIDs[0]

	// 获取requestId
	requestId := monitor.RequestID
	if requestId == "" {
		slog.Warn("缺少requestId参数")
		http.Error(w, "缺少requestId参数", http.StatusBadRequest)
		return
	}

	// 发送删除监控请求
	err = s.monitor.DeleteMonitoring(requestId)
	if err != nil {
		slog.Error("删除监控请求失败", "sliceID", sliceId, "requestID", requestId, "error", err)
		http.Error(w, "删除监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

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
		if errors.Is(err, mongo.ErrNilDocument) { // MongoDB为空文档
			slog.Warn("监控不存在", "monitorID", monitorId)
			http.Error(w, fmt.Sprintf("monitor不存在: %v", monitorId), http.StatusNotFound)
			return
		}

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
		if errors.Is(err, mongo.ErrNilDocument) { // MongoDB为空文档
			slog.Debug("monitor列表为空")
			w.WriteHeader(http.StatusOK)
			return
		}

		slog.Error("获取监控请求列表失败", "error", err)
		http.Error(w, "获取监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取监控请求列表成功", "count", len(monitors))
	encodeResponse(w, listMonitorResponse{Monitors: monitors})
}
