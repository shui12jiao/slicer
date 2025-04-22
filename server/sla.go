package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slicer/model"
)

func (s *Server) createSla(w http.ResponseWriter, r *http.Request) {
	var sla model.SLA
	if err := json.NewDecoder(r.Body).Decode(&sla); err != nil {
		http.Error(w, "请求解码失败", http.StatusBadRequest)
		return
	}

	// 检查值是否有效
	// TODO

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

func (s *Server) getSla(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取SLA请求", "method", r.Method, "url", r.URL.String())
	slaId := r.PathValue("slaId")
	if slaId == "" {
		slog.Error("缺少SLA ID")
		http.Error(w, "缺少SLA ID", http.StatusBadRequest)
		return
	}

	sla, err := s.store.GetSLA(slaId)
	if err != nil {
		if isNotFoundError(err) {
			slog.Warn("SLA不存在", "SLAID", slaId)
			http.Error(w, "SLA不存在", http.StatusNotFound)
			return
		}
		slog.Error("获取SLA失败", "SLAID", slaId, "error", err)
		http.Error(w, "获取SLA失败", http.StatusInternalServerError)
		return
	}
	// 返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sla); err != nil {
		slog.Error("响应编码失败", "SLAID", slaId, "sliceID", sla.SliceID, "error", err)
		http.Error(w, "响应编码失败", http.StatusInternalServerError)
		return
	}
	slog.Debug("获取SLA成功", "SLAID", sla.ID.Hex(), "sliceID", sla.SliceID)
}

func (s *Server) updateSla(w http.ResponseWriter, r *http.Request) {
	slog.Debug("更新SLA请求", "method", r.Method, "url", r.URL.String())
	slaId := r.PathValue("slaId")
	if slaId == "" {
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
	// TODO

	// 检查是否存在SLA
	curSLA, err := s.store.GetSLA(slaId)
	if err != nil {
		if isNotFoundError(err) {
			slog.Warn("SLA不存在", "SLAID", slaId)
			http.Error(w, "SLA不存在", http.StatusNotFound)
			return
		}
		slog.Error("获取SLA失败", "SLAID", slaId, "error", err)
		http.Error(w, "获取SLA失败", http.StatusInternalServerError)
		return
	}

	// 更新SLA
	err = curSLA.Update(sla)
	if err != nil {
		slog.Error("更新SLA失败", "SLAID", slaId, "error", err)
		http.Error(w, "更新SLA失败", http.StatusBadRequest)
		return
	}

	// 更新存储
	_, err = s.store.UpdateSLA(curSLA)
	if err != nil {
		slog.Error("更新SLA存储失败", "SLAID", slaId, "error", err)
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

func (s *Server) deleteSla(w http.ResponseWriter, r *http.Request) {
	slog.Debug("删除SLA请求", "method", r.Method, "url", r.URL.String())
	slaId := r.PathValue("slaId")
	if slaId == "" {
		slog.Error("缺少SLA ID")
		http.Error(w, "缺少SLA ID", http.StatusBadRequest)
		return
	}

	// 获取SLA
	sla, err := s.store.GetSLA(slaId)
	if err != nil {
		if isNotFoundError(err) {
			slog.Warn("SLA不存在", "SLAID", slaId)
			http.Error(w, "SLA不存在", http.StatusNotFound)
			return
		}
	}

	// 删除SLA
	if err := s.store.DeleteSLA(slaId); err != nil {
		slog.Error("删除SLA失败", "SLAID", slaId, "error", err)
		http.Error(w, "删除SLA失败", http.StatusInternalServerError)
		return
	}

	// controller删除slice
	s.controller.RemoveSlice(sla.SliceID)

	// 返回
	slog.Debug("删除SLA成功", "SLAID", sla.ID.Hex())
	w.WriteHeader(http.StatusOK)
}
