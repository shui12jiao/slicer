package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slicer/model"

	"github.com/go-chi/chi"
)

// updateDeploy godoc
// @Summary      更新Deploy
// @Description  更新Deploy资源,UpdateSlice的子功能
// @Tags         Slice
// @Accept       json
// @Produce      json
// @Param        slice_id path string true "切片ID"
// @Param        deploy body model.Deploy true "待更新的Deploy对象"
// @Success      200 {object} model.Deploy "更新成功，返回更新后的Deploy对象"
// @Failure      400 {string} string "请求参数错误或解码失败"
// @Failure      404 {string} string "未找到对应的Deploy"
// @Failure      500 {string} string "服务器内部错误"
// @Router       /slice/{slice_id}/deploy [put]
func (s *Server) updateDeploy(w http.ResponseWriter, r *http.Request) {
	slog.Debug("更新deploy请求", "method", r.Method, "url", r.URL.String(), "sliceID", chi.URLParam(r, "slice_id"))

	// 获取sliceID
	sliceID := chi.URLParam(r, "slice_id")
	if sliceID == "" {
		slog.Warn("缺少sliceID参数")
		http.Error(w, "缺少sliceID参数", http.StatusBadRequest)
		return
	}

	// body中获取deploy更新参数
	// 动态更新, 若值为空则不更新
	var deploy model.Deploy
	if err := json.NewDecoder(r.Body).Decode(&deploy); err != nil {
		http.Error(w, "请求解码失败", http.StatusBadRequest)
		return
	}

	// 检查值是否有效
	if err := deploy.Validate(); err != nil {
		slog.Error("非法值", "error", err)
		http.Error(w, fmt.Sprintf("非法值: %v", err), http.StatusBadRequest)
		return
	}

	deploy, err := s.service.UpdateDeploy(sliceID, deploy)
	if err != nil {
		slog.Error("更新deploy失败", "error", err, "sliceID", sliceID)
		if errors.Is(err, model.ErrSliceNotFound) {
			http.Error(w, fmt.Sprintf("Slice不存在: %v", sliceID), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("更新deploy失败: %v", err), http.StatusInternalServerError)
	}

	// 返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deploy); err != nil {
		slog.Error("响应编码失败", "sliceID", sliceID, "error", err)
		http.Error(w, "响应编码失败", http.StatusInternalServerError)
		return
	}
	slog.Debug("更新deploy成功", "sliceID", sliceID)
}
