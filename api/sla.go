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

// updateSLA godoc
// @Summary      更新SLA
// @Description  根据Slice ID更新SLA资源, UpdateSlice的子功能
// @Tags         Slice
// @Accept       json
// @Produce      json
// @Param        slice_id path string true "Slice ID"
// @Param        sla body model.SLA true "更新后的SLA对象"
// @Success      200 {object} model.SLA "更新成功返回对象"
// @Failure      400 {string} string "缺少Slice ID/请求解码失败/参数非法"
// @Failure      404 {string} string "Slice不存在"
// @Failure      500 {string} string "更新失败/响应编码失败"
// @Router       /slice/{slice_id}/sla [put]
func (s *Server) updateSLA(w http.ResponseWriter, r *http.Request) {
	slog.Debug("更新SLA请求", "method", r.Method, "url", r.URL.String(), "sliceID", chi.URLParam(r, "slice_id"))
	sliceID := chi.URLParam(r, "slice_id")
	if sliceID == "" {
		slog.Error("缺少Slice ID参数")
		http.Error(w, "缺少Slice ID参数", http.StatusBadRequest)
		return
	}

	var sla model.SLA
	if err := json.NewDecoder(r.Body).Decode(&sla); err != nil {
		http.Error(w, "请求解码失败", http.StatusBadRequest)
		return
	}

	// 检查值是否有效
	if err := sla.Validate(); err != nil {
		slog.Error("非法值", "error", err)
		http.Error(w, fmt.Sprintf("非法值: %v", err), http.StatusBadRequest)
		return
	}

	// 更新SLA
	sla, err := s.service.UpdateSLA(sliceID, sla)
	if err != nil {
		slog.Error("更新SLA失败", "error", err, "sliceID", sliceID)
		if errors.Is(err, model.ErrSliceNotFound) {
			http.Error(w, fmt.Sprintf("Slice不存在: %v", sliceID), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("更新SLA失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sla); err != nil {
		http.Error(w, "响应编码失败", http.StatusInternalServerError)
		return
	}
	slog.Debug("更新SLA成功", "sliceID", sliceID)
}
