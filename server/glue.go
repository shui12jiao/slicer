package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slicer/monitor"
)

// for Monarch
// 接受监控系统而非用户请求

// service orchestrator相关接口

// 用于响应监控系统request transltor的切片信息获取请求
// GET /service-orchestrator/slices/{sliceId}
type soGetSliceComponentsResponse struct {
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

func (s *Server) soGetSliceComponents(w http.ResponseWriter, r *http.Request) {
	sliceId := r.PathValue("sliceId")
	if sliceId == "" {
		http.Error(w, "缺少sliceId参数", http.StatusBadRequest)
		return
	}

	// 检查slice是否存在
	_, err := s.store.GetSliceBySliceID(sliceId)
	if err != nil {
		http.Error(w, "获取slice失败", http.StatusInternalServerError)
		return
	}

	pods, err := s.kubeclient.GetPods(s.config.Namespace)
	if err != nil {
		http.Error(w, "获取Pod失败", http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}
}

// 用于响应监控系统的so组件健康检查请求
// GET /service-orchestrator/api/health

type soCheckHealthResponse = monitor.Response

func (s *Server) soCheckHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encodeResponse(w, soCheckHealthResponse{
		Status:  "success",
		Message: "service orchestrator is healthy",
	})
}

// nfv orchestration相关接口

// 用于响应监控系统的mde安装请求
// POST /nfv-orchestrator/mde/install
type noMdeInstallRequest struct {
	SliceId string `json:"slice_id"`
}

func (s *Server) noMdeInstall(w http.ResponseWriter, r *http.Request) {
	// monarch的monitor manager组件中，process_slice_throughput_directive负责向no发送mdeinstall请求
	// 并未对directive（包含SliceComponents信息）进行解析

	// 从r中获取
	var req noMdeInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("请求解析失败: %v", err), http.StatusBadRequest)
		return
	}

	// 检查slice是否存在
	_, err := s.store.GetSliceBySliceID(req.SliceId)
	if err != nil {
		http.Error(w, "获取slice失败", http.StatusInternalServerError)
		return
	}

	// 渲染mde的yaml文件
	yaml, err := s.render.RenderMde([]string{req.SliceId})
	if err != nil {
		http.Error(w, fmt.Sprintf("渲染yaml失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 部署mde
	if err := s.kubeclient.Apply(yaml, s.config.MonitorNamespace); err != nil {
		http.Error(w, fmt.Sprintf("部署mde失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// 用于响应监控系统的mde卸载请求
// POST /nfv-orchestrator/mde/uninstall
// func (s *Server) noMdeUninstall(w http.ResponseWriter, r *http.Request) {
// 	// 无需实现，卸载直接由该系统完成，跳过监控系统
// }

// 用于响应监控系统的mde检查请求
// POST /nfv-orchestrator/mde/check

type noMdeCheckResponse struct {
	monitor.Response
	Output string `json:"output"`
}

func (s *Server) noMdeCheck(w http.ResponseWriter, r *http.Request) {
	// kubectl get svc -n open5gs -l app=monarch -o json | jq .items[].metadata.name
	svcs, err := s.kubeclient.GetServices(s.config.MonitorNamespace, "app=monarch")
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
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
}

// 用于响应监控系统的kpi计算组件安装请求
// POST /nfv-orchestrator/kpi-computation/install

type noKpiComputationInstallRequest = noMdeInstallRequest

func (s *Server) noKpiComputationInstall(w http.ResponseWriter, r *http.Request) {
	// monarch的monitor manager组件中，process_slice_throughput_directive负责向no发送mdeinstall请求
	// 并未对directive（包含SliceComponents信息）进行解析

	// 从r中获取
	var req noKpiComputationInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("请求解析失败: %v", err), http.StatusBadRequest)
		return
	}

	// 检查slice是否存在
	_, err := s.store.GetSliceBySliceID(req.SliceId)
	if err != nil {
		http.Error(w, "获取slice失败", http.StatusInternalServerError)
		return
	}

	// 渲染kpsc的yaml文件
	yaml, err := s.render.RenderKpiComp([]string{req.SliceId})
	if err != nil {
		http.Error(w, fmt.Sprintf("渲染yaml失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 部署kpic
	if err := s.kubeclient.Apply(yaml, s.config.MonitorNamespace); err != nil {
		http.Error(w, fmt.Sprintf("部署mde失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// 用于响应监控系统的kpi计算组件卸载请求
// POST /nfv-orchestrator/kpi-computation/uninstall
// func (s *Server) noKpiComputationUninstall(w http.ResponseWriter, r *http.Request) {
// 	// 无需实现，卸载直接由该系统完成，跳过监控系统
// }

// 用于响应监控系统的kpi计算组件检查请求
// POST /nfv-orchestrator/kpi-computation/check

type noKpiComputationCheckResponse = noMdeCheckResponse

func (s *Server) noKpiComputationCheck(w http.ResponseWriter, r *http.Request) {
	// kubectl get pods -n monarch -l app=monarch,component=kpi-calculator -o json | jq .items[].metadata.name
	pods, err := s.kubeclient.GetPods(s.config.MonitorNamespace, "app=monarch", "component=kpi-calculator")

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
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
}

// 用于响应监控系统的健康检查请求
// GET /nfv-orchestrator/api/health
type noCheckHealthResponse = monitor.Response

func (s *Server) noCheckHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encodeResponse(w, noCheckHealthResponse{
		Status:  "success",
		Message: "NFV Orchestrator is healthy",
	})
}
