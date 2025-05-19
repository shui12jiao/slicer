package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slicer/model"

	"github.com/go-chi/chi"
)

// createSla godoc
// @Summary      创建SLA
// @Description  接收SLA对象并创建新资源
// @Description  slice id对应切片必须存在
// @Tags         SLA
// @Accept       json
// @Produce      json
// @Param        sla body model.SLA true "SLA对象"
// @Success      200 {object} model.SLA "创建成功返回SLA对象"
// @Failure      400 {string} string "请求解码失败/SLA已存在/参数非法"
// @Failure      404 {string} string "切片不存在"
// @Failure      500 {string} string "存储失败/响应编码失败等服务器内部错误"
// @Router       /sla [post]
func (s *Server) createSla(w http.ResponseWriter, r *http.Request) {
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

	// 检查slice是否存在
	if _, err := s.store.GetSliceBySliceID(sla.SliceID); err != nil {
		if isNotFoundError(err) {
			slog.Warn("切片不存在", "sliceID", sla.SliceID)
			http.Error(w, "切片不存在", http.StatusNotFound)
			return
		}
		slog.Error("获取切片失败", "sliceID", sla.SliceID, "error", err)
		http.Error(w, "获取切片失败", http.StatusInternalServerError)
		return
	}

	// 检查是否有重复的SLA
	_, err := s.store.GetSLABySliceID(sla.SliceID)
	if err == nil {
		http.Error(w, "SLA已存在", http.StatusBadRequest)
		return
	}

	// 存储SLA
	sla, err = s.store.CreateSLA(sla)
	if err != nil {
		http.Error(w, "创建SLA失败", http.StatusInternalServerError)
		return
	}

	// 添加slice到controller
	s.controller.AddSlice(sla.SliceID)

	// 返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sla); err != nil {
		http.Error(w, "响应编码失败", http.StatusInternalServerError)
		return
	}
	slog.Debug("创建SLA成功", "SLAID", sla.ID.Hex())
}

// getSla godoc
// @Summary      获取单个SLA
// @Description  根据SLA ID获取资源详情
// @Tags         SLA
// @Accept       json
// @Produce      json
// @Param        slaID path string true "SLA ID"
// @Success      200 {object} model.SLA "获取成功"
// @Failure      400 {string} string "缺少SLA ID"
// @Failure      404 {string} string "SLA不存在"
// @Failure      500 {string} string "获取失败/响应编码失败"
// @Router       /sla/{sla_id} [get]
func (s *Server) getSla(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取SLA请求", "method", r.Method, "url", r.URL.String())
	slaID := chi.URLParam(r, "sla_id")
	if slaID == "" {
		slog.Error("缺少SLA ID")
		http.Error(w, "缺少SLA ID", http.StatusBadRequest)
		return
	}

	sla, err := s.store.GetSLA(slaID)
	if err != nil {
		if isNotFoundError(err) {
			slog.Warn("SLA不存在", "SLAID", slaID)
			http.Error(w, "SLA不存在", http.StatusNotFound)
			return
		}
		slog.Error("获取SLA失败", "SLAID", slaID, "error", err)
		http.Error(w, "获取SLA失败", http.StatusInternalServerError)
		return
	}
	// 返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sla); err != nil {
		slog.Error("响应编码失败", "SLAID", slaID, "sliceID", sla.SliceID, "error", err)
		http.Error(w, "响应编码失败", http.StatusInternalServerError)
		return
	}
	slog.Debug("获取SLA成功", "SLAID", sla.ID.Hex(), "sliceID", sla.SliceID)
}

// updateSla godoc
// @Summary      更新SLA
// @Description  根据SLA ID更新资源
// @Tags         SLA
// @Accept       json
// @Produce      json
// @Param        slaID path string true "SLA ID"
// @Param        sla body model.SLA true "更新后的SLA对象"
// @Success      200 {object} model.SLA "更新成功返回对象"
// @Failure      400 {string} string "缺少SLA ID/请求解码失败/参数非法"
// @Failure      404 {string} string "SLA不存在"
// @Failure      500 {string} string "更新失败/响应编码失败"
// @Router       /sla/{sla_id} [put]
func (s *Server) updateSla(w http.ResponseWriter, r *http.Request) {
	slog.Debug("更新SLA请求", "method", r.Method, "url", r.URL.String())
	slaID := chi.URLParam(r, "sla_id")
	if slaID == "" {
		slog.Error("缺少SLA ID")
		http.Error(w, "缺少SLA ID", http.StatusBadRequest)
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

	// 检查是否存在SLA
	curSLA, err := s.store.GetSLA(slaID)
	if err != nil {
		if isNotFoundError(err) {
			slog.Warn("SLA不存在", "SLAID", slaID)
			http.Error(w, "SLA不存在", http.StatusNotFound)
			return
		}
		slog.Error("获取SLA失败", "SLAID", slaID, "error", err)
		http.Error(w, "获取SLA失败", http.StatusInternalServerError)
		return
	}

	// 更新SLA
	err = curSLA.Update(sla)
	if err != nil {
		slog.Error("更新SLA失败", "SLAID", slaID, "error", err)
		http.Error(w, "更新SLA失败", http.StatusBadRequest)
		return
	}

	// 更新存储
	_, err = s.store.UpdateSLA(curSLA)
	if err != nil {
		slog.Error("更新SLA存储失败", "SLAID", slaID, "error", err)
		http.Error(w, "更新SLA存储失败", http.StatusInternalServerError)
		return
	}

	// 返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(curSLA); err != nil {
		http.Error(w, "响应编码失败", http.StatusInternalServerError)
		return
	}
	slog.Debug("更新SLA成功", "SLAID", curSLA.ID.Hex())
}

// deleteSla godoc
// @Summary      删除SLA
// @Description  根据SLA ID删除资源
// @Tags         SLA
// @Accept       json
// @Produce      json
// @Param        slaID path string true "SLA ID"
// @Success      200 {string} string "删除成功"
// @Failure      400 {string} string "缺少SLA ID"
// @Failure      404 {string} string "SLA不存在"
// @Failure      500 {string} string "删除失败"
// @Router       /sla/{sla_id} [delete]
func (s *Server) deleteSla(w http.ResponseWriter, r *http.Request) {
	slog.Debug("删除SLA请求", "method", r.Method, "url", r.URL.String())
	slaID := chi.URLParam(r, "sla_id")
	if slaID == "" {
		slog.Error("缺少SLA ID")
		http.Error(w, "缺少SLA ID", http.StatusBadRequest)
		return
	}

	// 获取SLA
	sla, err := s.store.GetSLA(slaID)
	if err != nil {
		if isNotFoundError(err) {
			slog.Warn("SLA不存在", "SLAID", slaID)
			http.Error(w, "SLA不存在", http.StatusNotFound)
			return
		}
	}

	// 删除SLA
	if err := s.store.DeleteSLA(slaID); err != nil {
		slog.Error("删除SLA失败", "SLAID", slaID, "error", err)
		http.Error(w, "删除SLA失败", http.StatusInternalServerError)
		return
	}

	// controller删除slice
	s.controller.RemoveSlice(sla.SliceID)

	// 返回
	slog.Debug("删除SLA成功", "SLAID", sla.ID.Hex())
	w.WriteHeader(http.StatusOK)
}

// listSla godoc
// @Summary      获取所有SLA
// @Description  获取系统内全部SLA列表
// @Tags         SLA
// @Accept       json
// @Produce      json
// @Success      200 {array} model.SLA "获取成功"
// @Failure      500 {string} string "获取失败/响应编码失败"
// @Router       /sla [get]
func (s *Server) listSla(w http.ResponseWriter, r *http.Request) {
	slog.Debug("列出SLA请求", "method", r.Method, "url", r.URL.String())
	slas, err := s.store.ListSLA()
	if err != nil {
		slog.Error("列出SLA失败", "error", err)
		http.Error(w, "列出SLA失败", http.StatusInternalServerError)
		return
	}

	// 返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(slas); err != nil {
		slog.Error("响应编码失败", "error", err)
		http.Error(w, "响应编码失败", http.StatusInternalServerError)
		return
	}
	slog.Debug("列出SLA成功")
}
