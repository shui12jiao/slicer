package api

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
	Frequency time.Duration `json:"frequency" swaggertype:"primitive,integer" format:"nanoseconds" example:"1000000000"`

	// 切片列表
	Slices []string `json:"slices"`
	// 策略名称列表
	Strategies []string `json:"strategies"`
	// 使用策略
	UsedStrategy string `json:"used_strategy"`
}

// getController godoc
// @Summary      获取控制器状态
// @Description  返回控制器运行状态、策略列表及当前生效策略
// @Tags         Controller
// @Accept       json
// @Produce      json
// @Success      200 {object} ControllerResponse "成功获取控制器状态"
// @Failure      500 {string} string "响应编码失败"
// @Router       /controller [get]
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
	Frequency *time.Duration `json:"frequency" swaggertype:"primitive,integer" format:"nanoseconds" example:"1000000000"`
	// 使用策略
	UsedStrategy *string `json:"used_strategy"`
	// 切片列表
	AddSlices    []string `json:"add_slices,omitempty"`    // 添加切片列表
	RemoveSlices []string `json:"remove_slices,omitempty"` // 移除切片
}

// updateController godoc
// @Summary      更新控制器配置
// @Description  动态更新运行状态、控制频率或切换调度策略，声明式
// @Tags         Controller
// @Accept       json
// @Produce      json
// @Param        config body UpdateControllerRequest true "更新配置参数"
// @Success      200 "配置更新成功"
// @Failure      400 {string} string "请求解析失败/策略不存在"
// @Failure      500 {string} string "内部配置更新失败"
// @Router       /controller [patch]
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
		running := *req.Running
		if running == controller.IsRunning() {
			slog.Warn("控制器状态未变化, 跳过更新", "running", running)
		} else {
			if running {
				controller.Start()
			} else {
				controller.Stop()
			}
			slog.Info("控制器状态更新", "running", running)
		}
	}

	// 处理频率
	if req.Frequency != nil {
		controller.SetFrequency(*req.Frequency)
		slog.Info("控制器频率更新", "frequency", *req.Frequency)
	}

	// 处理策略
	if req.UsedStrategy != nil {
		strategy := controller.GetStrategy()
		if strategy != nil && strategy.Name() == *req.UsedStrategy {
			slog.Warn("策略未变化, 跳过", "策略名称", *req.UsedStrategy)
			return
		}
		s := controller.GetStrategyByName(*req.UsedStrategy)
		if s == nil {
			slog.Error("策略不存在, 无法使用", "策略名称", *req.UsedStrategy)
			http.Error(w, "策略不存在", http.StatusBadRequest)
			return
		}
		controller.SetStrategy(s)
		slog.Info("控制器策略更新", "策略名称", *req.UsedStrategy)
	}

	// 处理切片列表
	if len(req.AddSlices) > 0 {
		controller.AddSlice(req.AddSlices)
		slog.Info("添加切片", "切片ID列表", req.AddSlices)
	}
	if len(req.RemoveSlices) > 0 {
		controller.RemoveSlice(req.RemoveSlices)
		slog.Info("移除切片", "切片ID列表", req.RemoveSlices)
	}

	// 返回成功响应
	w.WriteHeader(http.StatusOK)
}

type triggerRequest struct {
	Slices []string `json:"slices"`
}

// triggerNow godoc
// @Summary      触发控制器立即运行
// @Description  触发控制器立即运行，可以指定切片ID或全部切片(传入空或者手动传入全部切片ID)
// @Tags         Controller
// @Accept       json
// @Produce      json
// @Param        slices body triggerRequest true "切片ID列表, 如果为空则触发全部切片"
// @Success      200 "触发成功"
// @Failure      400 {string} string "请求解析失败"
// @Failure      500 {string} string "触发控制器失败"
// @Router       /controller/trigger [post]
func (s *Server) triggerNow(w http.ResponseWriter, r *http.Request) {
	slog.Debug("触发控制器立即运行请求", "method", r.Method, "url", r.URL.String())
	// 触发控制器立即运行
	var req triggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("解析请求失败", "error", err)
		http.Error(w, "解析请求失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.controller.Trigger(req.Slices); err != nil {
		slog.Error("触发控制器失败", "error", err)
		http.Error(w, "触发控制器失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	w.WriteHeader(http.StatusOK)
}
