package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slicer/model"
)

// getSupportedKpis godoc
// @Summary      获取支持监控的KPI列表
// @Description  返回系统支持的所有KPI指标定义
// @Tags         Monitor
// @Accept       json
// @Produce      json
// @Success      200 {array} model.SupportedKpi "成功获取KPI列表"
// @Failure      500 {string} string "服务器内部错误（获取失败）"
// @Router       /monitor/supported_kpis [get]
func (s *Server) getSupportedKpis(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取支持的KPI请求", "method", r.Method, "url", r.URL.String())

	kpis, err := s.monitor.GetSupportedKpis()
	if err != nil {
		slog.Error("获取支持的KPI失败", "error", err)
		http.Error(w, "获取支持的KPI失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取支持的KPI成功", "count", len(kpis))
	encodeResponse(w, kpis)
}

// createMonitor godoc
// @Summary      创建监控资源（内置部署）
// @Description  创建监控请求并部署到Kubernetes集群，包含MDE和KPI组件
// @Tags         Monitor
// @Accept       json
// @Produce      json
// @Param        monitor body model.Monitor true "监控配置对象"
// @Success      201 {object} model.Monitor "创建成功返回监控对象"
// @Failure      400 {string} string "请求解码失败/参数验证失败/Slice不存在"
// @Failure      404 {string} string "关联Slice不存在"
// @Failure      500 {string} string "渲染YAML失败/部署失败/存储失败"
// @Router       /monitor [post]
func (s *Server) createMonitor(w http.ResponseWriter, r *http.Request) {
	slog.Debug("创建监控请求", "method", r.Method, "url", r.URL.String())

	// 解析请求
	var monitor model.Monitor
	if err := json.NewDecoder(r.Body).Decode(&monitor); err != nil {
		slog.Warn("请求解码失败", "error", err)
		http.Error(w, "请求解码失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 检查请求参数
	if err := monitor.Validate(); err != nil {
		slog.Warn("请求验证失败", "error", err)
		http.Error(w, "请求验证失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 获取sliceId
	sliceId := monitor.KPI.SubCounter.SubCounterIDs[0]
	if sliceId == "" {
		slog.Debug("无sliceID参数, 默认进行全部监控")
	} else {
		slog.Debug("获取sliceId参数", "sliceID", sliceId)
		// 检查sliceId是否存在
		if _, err := s.store.GetSliceBySliceID(sliceId); err != nil {
			if isNotFoundError(err) { // MongoDB为空文档
				slog.Warn("要求监控的sliceId不存在", "sliceID", sliceId)
				http.Error(w, fmt.Sprintf("要求监控的sliceId不存在: %v", sliceId), http.StatusBadRequest)
				return
			}

			slog.Error("获取sliceId失败", "sliceID", sliceId, "error", err)
			http.Error(w, "获取sliceId失败: "+err.Error(), http.StatusInternalServerError)
			return
		}
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
	// 注意这里使用了s.config.Namespace, 使用metrics+annotations的方式使prometheus进行抓取
	// 如果使用了crd: service monitor, 需要使用s.config.MonitorNamespace(service中没有定义metrics, 直接使用service monitor似乎不工作因为port: metrics没有定义)
	if err := s.kubeclient.ApplyMDE(yamlMde); err != nil {
		slog.Error("部署MDE失败", "sliceID", sliceId, "error", err)
		http.Error(w, "部署MDE失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 部署KPI
	if err := s.kubeclient.ApplyKpic(yamlKpi); err != nil {
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
	encodeResponse(w, monitor)
}

// createMonitorExternal godoc
// @Summary      创建监控资源（外部服务）
// @Description  通过Monarch外部服务提交监控请求
// @Tags         Monitor
// @Accept       json
// @Produce      json
// @Param        monitor body model.Monitor true "监控配置对象"
// @Success      201 {object} model.Monitor "创建成功返回监控对象"
// @Failure      400 {string} string "请求解码失败/参数验证失败/Slice不存在"
// @Failure      500 {string} string "提交外部请求失败/存储失败"
// @Router       /monitor/external [post]
func (s *Server) createMonitorExternal(w http.ResponseWriter, r *http.Request) {
	slog.Debug("创建监控请求", "method", r.Method, "url", r.URL.String())

	// 解析请求
	var monitor model.Monitor
	if err := json.NewDecoder(r.Body).Decode(&monitor); err != nil {
		slog.Warn("请求解码失败", "error", err)
		http.Error(w, "请求解码失败: "+err.Error(), http.StatusBadRequest)
		return
	}

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
	encodeResponse(w, monitor)
}

// deleteMonitor godoc
// @Summary      删除监控资源（内置部署）
// @Description  根据ID删除监控资源并清理Kubernetes组件
// @Tags         Monitor
// @Accept       json
// @Produce      json
// @Param        monitorId path string true "监控记录ID"
// @Success      204 "资源删除成功"
// @Failure      400 {string} string "缺少监控ID参数"
// @Failure      404 {string} string "监控记录不存在"
// @Failure      500 {string} string "YAML渲染失败/组件删除失败/存储删除失败"
// @Router       /monitor/slice/{monitorId} [delete]
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
		if isNotFoundError(err) { // MongoDB为空文档
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
	err = s.kubeclient.DeleteMDE(yaml) // 注意这里使用了s.config.Namespace, 和上面创建时必须一致
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
	err = s.kubeclient.DeleteKpic(yaml)
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

// deleteMonitorExternal godoc
// @Summary      删除监控资源（外部服务）
// @Description  通过Monarch外部服务删除监控请求
// @Tags         Monitor
// @Accept       json
// @Produce      json
// @Param        monitorId path string true "监控记录ID"
// @Param        requestId query string true "外部服务请求ID"
// @Success      204 "资源删除成功"
// @Failure      400 {string} string "缺少监控ID或请求ID"
// @Failure      404 {string} string "监控记录不存在"
// @Failure      500 {string} string "外部服务删除失败/存储删除失败"
// @Router       /monitor/external/{monitorId} [delete]
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
		if isNotFoundError(err) { // MongoDB为空文档
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

// getMonitor godoc
// @Summary      获取单个监控配置
// @Description  根据监控ID获取详细配置信息
// @Tags         Monitor
// @Accept       json
// @Produce      json
// @Param        monitorId path string true "监控记录ID"
// @Success      200 {object} model.Monitor "获取成功"
// @Failure      400 {string} string "缺少监控ID参数"
// @Failure      404 {string} string "监控记录不存在"
// @Failure      500 {string} string "获取数据失败"
// @Router       /monitor/slice/{monitorId} [get]
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
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("监控不存在", "monitorID", monitorId)
			http.Error(w, fmt.Sprintf("monitor不存在: %v", monitorId), http.StatusNotFound)
			return
		}

		slog.Error("获取监控请求失败", "monitorID", monitorId, "error", err)
		http.Error(w, "获取监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取监控请求成功", "monitorID", monitorId)
	encodeResponse(w, monitor)
}

// listMonitor godoc
// @Summary      获取所有监控配置
// @Description  获取系统中存在的所有监控配置列表
// @Tags         Monitor
// @Accept       json
// @Produce      json
// @Success      200 {array} model.Monitor "获取成功"
// @Failure      500 {string} string "获取数据失败"
// @Router       /monitor [get]
func (s *Server) listMonitor(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取监控请求列表", "method", r.Method, "url", r.URL.String())

	monitors, err := s.store.ListMonitor()
	if err != nil { // 为空时list不会返回错误
		// if isNotFoundError(err) { // MongoDB为空文档
		// 	slog.Debug("monitor列表为空")
		// 	w.WriteHeader(http.StatusOK)
		// 	return
		// }

		slog.Error("获取监控请求列表失败", "error", err)
		http.Error(w, "获取监控请求失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取监控请求列表成功", "count", len(monitors))
	encodeResponse(w, monitors)
}
