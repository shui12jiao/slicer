package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slicer/monitor"
)

// for Monarch

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
func (s *Server) noMdeInstall(w http.ResponseWriter, r *http.Request) {
	// TODO
}

// 用于响应监控系统的mde卸载请求
// POST /nfv-orchestrator/mde/uninstall
func (s *Server) noMdeUninstall(w http.ResponseWriter, r *http.Request) {
	// TODO
}

// 用于响应监控系统的mde检查请求
// POST /nfv-orchestrator/mde/check
func (s *Server) noMdeCheck(w http.ResponseWriter, r *http.Request) {
	// TODO
}

// 用于响应监控系统的kpi计算组件安装请求
// POST /nfv-orchestrator/kpi-computation/install
func (s *Server) noKpiComputationInstall(w http.ResponseWriter, r *http.Request) {
	// TODO
}

// 用于响应监控系统的kpi计算组件卸载请求
// POST /nfv-orchestrator/kpi-computation/uninstall
func (s *Server) noKpiComputationUninstall(w http.ResponseWriter, r *http.Request) {
	// TODO
}

// 用于响应监控系统的kpi计算组件检查请求
// POST /nfv-orchestrator/kpi-computation/check
func (s *Server) noKpiComputationCheck(w http.ResponseWriter, r *http.Request) {
	// TODO
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
