package server

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/exp/slog"
)

type ControllerResponse struct {
	// 运行状态
	Running bool `json:"running"`
	// 控制频率
	Frequency time.Duration `json:"frequency"`

	// 切片列表
	Slices []string `json:"slices"`
	// 策略名称列表
	Strategies []string `json:"strategies"`
	// 使用策略
	UsedStrategy string `json:"used_strategy"`
}

func (s *Server) getController(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取控制器状态请求", "method", r.Method, "url", r.URL.String())
	// 获取控制器的状态
	controller := s.controller
	response := ControllerResponse{
		Running:   controller.IsRunning(),
		Frequency: controller.GetFrequency(),
		Slices:    controller.ListSlices(), //返回控制器中所有切片的ID
		Strategies: func() []string { //返回所有策略的名称
			strategies := controller.ListStrategy()
			names := make([]string, len(strategies))
			for i, strategy := range strategies {
				names[i] = strategy.Name()
			}
			return names
		}(),
		UsedStrategy: func() string { //返回当前使用的策略名称, 如果没有使用任何策略，则返回空字符串
			strategy := controller.GetStrategy()
			if strategy != nil {
				return strategy.Name()
			}
			return ""
		}(),
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// 动态更新控制器的状态
type UpdateControllerRequest struct {
	// 运行状态
	Running *bool `json:"running"`
	// 控制频率
	Frequency *time.Duration `json:"frequency"`
	// 使用策略
	UsedStrategy *string `json:"used_strategy"`
}

func (s *Server) updateController(w http.ResponseWriter, r *http.Request) {
	slog.Debug("更新控制器状态请求", "method", r.Method, "url", r.URL.String())
	var req UpdateControllerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("解析请求失败", "error", err)
		http.Error(w, "解析请求失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 更新控制器状态
	controller := s.controller
	// 处理运行状态
	if req.Running != nil {
		if *req.Running == controller.IsRunning() {
			slog.Warn("控制器状态未变化, 跳过更新", "running", *req.Running)
		} else {
			if *req.Running {
				controller.Start()
			} else {
				controller.Stop()
			}
			slog.Info("控制器状态更新", "running", *req.Running)
		}
	}
	// 处理频率
	if req.Frequency != nil {
		controller.SetFrequency(*req.Frequency)
		slog.Info("控制器频率更新", "frequency", *req.Frequency)
	}
	// 处理切片列表
	if req.UsedStrategy != nil {
		strategy := controller.GetStrategy()
		if strategy != nil && strategy.Name() == *req.UsedStrategy {
			slog.Warn("策略已存在, 跳过注册", "策略名称", *req.UsedStrategy)
			return
		}
		s := controller.GetStrategyByName(*req.UsedStrategy)
		if s == nil {
			slog.Error("策略不存在, 无法注册", "策略名称", *req.UsedStrategy)
			http.Error(w, "策略不存在", http.StatusBadRequest)
			return
		}
		controller.SetStrategy(s)
	}

	// 返回成功响应
	w.WriteHeader(http.StatusOK)
}
