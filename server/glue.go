package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slicer/monitor"
)

// for Monarch
// 接受监控系统而非用户请求

// Service Orchestrator 接口
//================================================================================

type soGetSliceComponentsResponse struct {
	// Example:
	// {
	// 	"pods": [
	// 	  {
	// 		"name": "open5gs-smf1-000001-67cf5ccccd-rzvl6",
	// 		"nf": "smf",
	// 		"nss": "edge",
	// 		"pod_ip": ""
	// 	  },
	// 	  {
	// 		"name": "open5gs-upf1-000001-7f6b8444f-grp98",
	// 		"nf": "upf",
	// 		"nss": "edge",
	// 		"pod_ip": ""
	// 	  }
	// 	],
	// 	"status": "success"
	//   }
	Pods []struct {
		Name  string `json:"name"`
		NF    string `json:"nf"`
		NSS   string `json:"nss"`
		PodIP string `json:"pod_ip"`
	} `json:"pods"`
	monitor.Response
}

// soGetSliceComponents godoc
// @Summary      获取切片组件信息
// @Description  查询指定切片下的NFV组件Pod详细信息（面向监控系统）
// @Tags         Service Orchestrator
// @Accept       json
// @Produce      json
// @Param        sliceId path string true "切片唯一标识符" Example(edge01)
// @Success      200 {object} soGetSliceComponentsResponse
// @Failure      400 {object} monitor.Response "参数校验失败"
// @Failure      404 {object} monitor.Response "切片不存在"
// @Failure      500 {object} monitor.Response "服务器内部错误"
// @Router       /service-orchestrator/slices/{sliceId} [get]
func (s *Server) soGetSliceComponents(w http.ResponseWriter, r *http.Request) {
	slog.Debug("SO: 处理切片组件请求", "method", r.Method, "path", r.URL.Path)
	sliceId := r.PathValue("sliceId")
	if sliceId == "" {
		slog.Warn("SO: 缺少sliceId参数")
		http.Error(w, "缺少sliceId参数", http.StatusBadRequest)
		return
	}

	// 检查slice是否存在
	_, err := s.store.GetSliceBySliceID(sliceId)
	if err != nil {
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("SO: slice不存在", "sliceID", sliceId)
			http.Error(w, fmt.Sprintf("slice不存在: %v", sliceId), http.StatusNotFound)
			return
		}

		slog.Error("SO: 获取slice失败", "sliceID", sliceId, "error", err)
		http.Error(w, fmt.Sprintf("获取slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	pods, err := s.kubeclient.GetPods(s.config.Namespace)
	if err != nil {
		slog.Error("SO: 获取Pods失败", "namespace", s.config.Namespace, "error", err)
		http.Error(w, fmt.Sprintf("获取Pods失败: %v", err), http.StatusInternalServerError)
		return
	}

	var resp soGetSliceComponentsResponse
	for _, pod := range pods {
		labels := pod.Labels
		// ex: name=smf1-000001, nf=smf
		if labels["name"] == labels["nf"]+sliceId {
			resp.Pods = append(resp.Pods, struct {
				Name  string `json:"name"`
				NF    string `json:"nf"`
				NSS   string `json:"nss"`
				PodIP string `json:"pod_ip"`
			}{
				Name:  pod.Name,
				NF:    labels["nf"],
				NSS:   "edge",
				PodIP: pod.Status.PodIP,
			})
		}
	}
	resp.Status = "success"

	//设置响应头
	w.Header().Set("Content-Type", "application/json")
	//编码响应
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("SO: 响应编码失败", "error", err)
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}
	slog.Debug("SO: 切片组件请求处理完成", "sliceID", sliceId, "podsCount", len(resp.Pods))
}

// soCheckHealth godoc
// @Summary      服务健康检查
// @Description  验证Service Orchestrator组件运行状态
// @Tags         Service Orchestrator
// @Accept       json
// @Produce      json
// @Success      200 {object} monitor.Response "服务正常运行"
// @Router       /service-orchestrator/api/health [get]
func (s *Server) soCheckHealth(w http.ResponseWriter, r *http.Request) {
	slog.Debug("SO: 处理健康检查请求")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encodeResponse(w, monitor.Response{
		Status:  "success",
		Message: "service orchestrator is healthy",
	})
	slog.Debug("SO: 健康检查完成")
}

// NFV Orchestrator 接口
//================================================================================

type noMdeInstallRequest struct {
	// monitornig_manager向no发送的请求实际为空, 故使用omitempty
	// 当sliceId为空时,暂且认为是监控全部slice
	SliceId string `json:"slice_id,omitempty"`
}

// noMdeInstall godoc
// @Summary      安装监控数据采集器（MDE）
// @Description  根据切片ID部署Prometheus exporter组件到指定命名空间
// @Tags         NFV Orchestrator
// @Accept       json
// @Produce      json
// @Param        body body noMdeInstallRequest true "请求参数"
// @Success      200 {object} monitor.Response "MDE安装成功"
// @Failure      400 {object} monitor.Response "请求参数不合法"
// @Failure      404 {object} monitor.Response "切片不存在"
// @Failure      500 {object} monitor.Response "渲染YAML失败/K8s部署失败"
// @Router       /nfv-orchestrator/mde/install [post]
func (s *Server) noMdeInstall(w http.ResponseWriter, r *http.Request) {
	slog.Debug("NO: 处理MDE安装请求", "method", r.Method, "path", r.URL.Path)
	// monarch的monitor manager组件中，process_slice_throughput_directive负责向no发送mdeinstall请求
	// 并未对directive（包含SliceComponents信息）进行解析

	// 对监控系统暂时的单独处理
	// 暂时认为如果sliceId为空, 则监控全部slice
	if r.Body == http.NoBody {
		// 处理空请求体的逻辑
		slog.Info("NO: 接收到空请求体，安装全局MDE")

		// 渲染mde的yaml文件
		yaml, err := s.render.RenderMde("")
		if err != nil {
			slog.Error("NO: 渲染全局MDE yaml失败", "error", err)
			http.Error(w, fmt.Sprintf("渲染yaml失败: %v", err), http.StatusInternalServerError)
			return
		}

		// 部署mde
		if err := s.kubeclient.ApplyMDE(yaml); err != nil {
			slog.Error("NO: 部署全局MDE失败", "namespace", s.config.MonitorNamespace, "error", err)
			http.Error(w, fmt.Sprintf("部署mde失败: %v", err), http.StatusInternalServerError)
			return
		}

		// 返回响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		slog.Info("NO: 全局MDE安装成功")
		return
	}

	// 从r中获取
	var req noMdeInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("NO: 请求解析失败", "error", err)
		http.Error(w, fmt.Sprintf("请求解析失败: %v", err), http.StatusBadRequest)
		return
	}

	// 检查slice是否存在
	_, err := s.store.GetSliceBySliceID(req.SliceId)
	if err != nil {
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("NO: slice不存在", "sliceID", req.SliceId)
			http.Error(w, fmt.Sprintf("slice不存在: %v", req.SliceId), http.StatusNotFound)
			return
		}

		slog.Error("NO: 获取slice失败", "sliceID", req.SliceId, "error", err)
		http.Error(w, "获取slice失败", http.StatusInternalServerError)
		return
	}

	// 渲染mde的yaml文件
	yaml, err := s.render.RenderMde(req.SliceId)
	if err != nil {
		slog.Error("NO: 渲染MDE yaml失败", "sliceID", req.SliceId, "error", err)
		http.Error(w, fmt.Sprintf("渲染yaml失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 部署mde
	if err := s.kubeclient.ApplyMDE(yaml); err != nil {
		slog.Error("NO: 部署MDE失败", "sliceID", req.SliceId, "namespace", s.config.MonitorNamespace, "error", err)
		http.Error(w, fmt.Sprintf("部署mde失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	slog.Info("NO: MDE安装成功", "sliceID", req.SliceId)
}

// noMdeUninstall godoc
// @Summary      卸载监控数据采集器（MDE）
// @Description  移除当前命名空间下的Prometheus exporter组件
// @Tags         NFV Orchestrator
// @Accept       json
// @Produce      json
// @Success      200 {object} monitor.Response "MDE卸载成功"
// @Failure      500 {object} monitor.Response "YAML渲染失败/K8s删除失败"
// @Router       /nfv-orchestrator/mde/uninstall [post]
func (s *Server) noMdeUninstall(w http.ResponseWriter, r *http.Request) {
	// 实际请求参数为空, 直接由监控系统完成卸载

	// 删除MDE
	yaml, err := s.render.RenderMde("")
	if err != nil {
		slog.Error("NO: 渲染MDE yaml失败", "error", err)
		http.Error(w, "渲染yaml失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = s.kubeclient.DeleteMDE(yaml)
	if err != nil {
		slog.Error("NO: 删除MDE失败", "namespace", s.config.MonitorNamespace, "error", err)
		http.Error(w, "删除MDE失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 编码响应
	w.WriteHeader(http.StatusOK)
	encodeResponse(w, monitor.Response{
		Status:  "success",
		Message: "MDE 删除成功",
	},
	)
	slog.Debug("NO: MDE卸载完成", "namespace", s.config.MonitorNamespace)
}

type noMdeCheckResponse struct {
	monitor.Response
	Output string `json:"output"`
}

// noMdeCheck godoc
// @Summary      检查MDE运行状态
// @Description  验证监控数据采集器的服务端点是否就绪
// @Tags         NFV Orchestrator
// @Accept       json
// @Produce      json
// @Success      200 {object} noMdeCheckResponse "MDE服务列表"
// @Failure      500 {object} noMdeCheckResponse "服务查询失败"
// @Router       /nfv-orchestrator/mde/check [post]
func (s *Server) noMdeCheck(w http.ResponseWriter, r *http.Request) {
	slog.Debug("NO: 开始MDE检查")
	// kubectl get svc -n open5gs -l app=monarch -o json | jq .items[].metadata.name
	svcs, err := s.kubeclient.GetServices(s.config.MonitorNamespace, "app=monarch")
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		slog.Error("NO: 获取MDE服务失败", "namespace", s.config.MonitorNamespace, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		encodeResponse(w, noMdeCheckResponse{
			Response: monitor.Response{
				Status:  "error",
				Message: "MDE 测试失败",
			},
			Output: fmt.Sprintf("获取服务失败: %v", err),
		})
		return
	}

	// 只返回服务名称
	var serviceNames []string
	for _, svc := range svcs {
		serviceNames = append(serviceNames, svc.Name)
	}

	// 编码响应
	w.WriteHeader(http.StatusOK)
	encodeResponse(w, noMdeCheckResponse{
		Response: monitor.Response{
			Status:  "success",
			Message: "MDE 测试成功",
		},
		Output: fmt.Sprintf("获取服务成功: %v", serviceNames),
	})
	slog.Debug("NO: MDE检查完成", "servicesCount", len(serviceNames))
}

type noKpiComputationInstallRequest = noMdeInstallRequest

// noKpiComputationInstall godoc
// @Summary      安装KPI计算组件
// @Description  部署实时KPI计算引擎到监控命名空间
// @Tags         NFV Orchestrator
// @Accept       json
// @Produce      json
// @Param        body body noKpiComputationInstallRequest true "请求参数"
// @Success      200 {object} monitor.Response "KPI组件安装成功"
// @Failure      400 {object} monitor.Response "参数校验失败"
// @Failure      500 {object} monitor.Response "YAML渲染/K8s部署失败"
// @Router       /nfv-orchestrator/kpi-computation/install [post]
func (s *Server) noKpiComputationInstall(w http.ResponseWriter, r *http.Request) {
	slog.Debug("NO: 处理KPI计算组件安装请求", "method", r.Method, "path", r.URL.Path)
	// monarch的monitor manager组件中，process_slice_throughput_directive负责向no发送mdeinstall请求
	// 并未对directive（包含SliceComponents信息）进行解析

	// 对监控系统暂时的单独处理
	// 暂时认为如果sliceId为空, 则监控全部slice
	if r.Body == http.NoBody {
		// 处理空请求体的逻辑
		slog.Info("NO: 接收到空请求体，安装全局KPI计算组件")

		// 渲染kpsc的yaml文件
		yaml, err := s.render.RenderKpiCalc("")
		if err != nil {
			slog.Error("NO: 渲染全局KPI计算组件yaml失败", "error", err)
			http.Error(w, fmt.Sprintf("渲染yaml失败: %v", err), http.StatusInternalServerError)
			return
		}

		// yaml写入到tmp.yaml for test
		// os.WriteFile("tmp.yaml", []byte(yaml), 0644)

		// 部署kpic
		if err := s.kubeclient.ApplyKpic(yaml); err != nil {
			slog.Error("NO: 部署全局KPI计算组件失败", "namespace", s.config.MonitorNamespace, "error", err)
			http.Error(w, fmt.Sprintf("部署kpic失败: %v", err), http.StatusInternalServerError)
			return
		}

		// 返回响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		slog.Info("NO: 全局KPI计算组件安装成功")
		return
	}

	// 从r中获取
	var req noKpiComputationInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("NO: 请求解析失败", "error", err)
		http.Error(w, fmt.Sprintf("请求解析失败: %v", err), http.StatusBadRequest)
		return
	}

	// 检查slice是否存在
	_, err := s.store.GetSliceBySliceID(req.SliceId)
	if err != nil {
		slog.Error("NO: 获取slice失败", "sliceID", req.SliceId, "error", err)
		http.Error(w, "获取slice失败", http.StatusInternalServerError)
		return
	}

	// 渲染kpsc的yaml文件
	yaml, err := s.render.RenderKpiCalc(req.SliceId)
	if err != nil {
		slog.Error("NO: 渲染KPI计算组件yaml失败", "sliceID", req.SliceId, "error", err)
		http.Error(w, fmt.Sprintf("渲染yaml失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 部署kpic
	if err := s.kubeclient.ApplyKpic(yaml); err != nil {
		slog.Error("NO: 部署KPI计算组件失败", "sliceID", req.SliceId, "namespace", s.config.MonitorNamespace, "error", err)
		http.Error(w, fmt.Sprintf("部署mde失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	slog.Info("NO: KPI计算组件安装成功", "sliceID", req.SliceId)
}

// noKpiComputationUninstall godoc
// @Summary      卸载KPI计算组件
// @Description  移除KPI计算引擎相关资源
// @Tags         NFV Orchestrator
// @Accept       json
// @Produce      json
// @Success      200 {object} monitor.Response "KPI组件卸载成功"
// @Failure      500 {object} monitor.Response "YAML渲染/K8s删除失败"
// @Router       /nfv-orchestrator/kpi-computation/uninstall [post]
func (s *Server) noKpiComputationUninstall(w http.ResponseWriter, r *http.Request) {
	// 实际请求参数为空, 直接由监控系统完成卸载

	// 删除KPI
	yaml, err := s.render.RenderKpiCalc("")
	if err != nil {
		slog.Error("NO: 渲染KPI yaml失败", "error", err)
		http.Error(w, "渲染yaml失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = s.kubeclient.DeleteKpic(yaml)
	if err != nil {
		slog.Error("NO: 删除KPI失败", "namespace", s.config.MonitorNamespace, "error", err)
		http.Error(w, "删除KPI失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 编码响应
	w.WriteHeader(http.StatusOK)
	encodeResponse(w, monitor.Response{
		Status:  "success",
		Message: "KPI 删除成功",
	},
	)
	slog.Debug("NO: KPI计算组件卸载完成", "namespace", s.config.MonitorNamespace)
}

type noKpiComputationCheckResponse = noMdeCheckResponse

// noKpiComputationCheck godoc
// @Summary      检查KPI组件状态
// @Description  验证KPI计算引擎Pod的运行状态
// @Tags         NFV Orchestrator
// @Accept       json
// @Produce      json
// @Success      200 {object} noKpiComputationCheckResponse "Pod状态列表"
// @Failure      500 {object} noKpiComputationCheckResponse "Pod查询失败"
// @Router       /nfv-orchestrator/kpi-computation/check [post]
func (s *Server) noKpiComputationCheck(w http.ResponseWriter, r *http.Request) {
	slog.Debug("NO: 开始KPI计算组件检查")
	// kubectl get pods -n monarch -l app=monarch,component=kpi-calculator -o json | jq .items[].metadata.name
	pods, err := s.kubeclient.GetPods(s.config.MonitorNamespace, "app=monarch", "component=kpi-calculator")

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		slog.Error("NO: 获取KPI计算Pod失败", "namespace", s.config.MonitorNamespace, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		encodeResponse(w, noKpiComputationCheckResponse{
			Response: monitor.Response{
				Status:  "error",
				Message: "KPI computation 测试失败",
			},
			Output: fmt.Sprintf("获取服务失败: %v", err),
		})
		return
	}

	// 只返回服务名称
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}

	// 编码响应
	w.WriteHeader(http.StatusOK)
	encodeResponse(w, noKpiComputationCheckResponse{
		Response: monitor.Response{
			Status:  "success",
			Message: "KPI computation 测试成功",
		},
		Output: fmt.Sprintf("获取Pod成功: %v", podNames),
	})
	slog.Debug("NO: KPI计算组件检查完成", "podsCount", len(podNames))
}

type noCheckHealthResponse = monitor.Response

// noCheckHealth godoc
// @Summary      NFV Orchestrator健康检查
// @Description  验证NFV编排组件的运行状态
// @Tags         NFV Orchestrator
// @Accept       json
// @Produce      json
// @Success      200 {object} noCheckHealthResponse "服务健康状态"
// @Router       /nfv-orchestrator/api/health [get]
func (s *Server) noCheckHealth(w http.ResponseWriter, r *http.Request) {
	slog.Debug("NO: 处理健康检查请求")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encodeResponse(w, noCheckHealthResponse{
		Status:  "success",
		Message: "NFV Orchestrator is healthy",
	})
	slog.Debug("NO: 健康检查完成")
}
