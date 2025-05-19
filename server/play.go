package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slicer/model"

	"github.com/go-chi/chi"
)

// createPlay godoc
// @Summary      创建Play资源
// @Description  接收Play对象并创建新资源，同时部署到Kubernetes集群
// @Tags         Play
// @Accept       json
// @Produce      json
// @Param        play body model.Play true "Play对象"
// @Success      200 {object} model.Play "创建成功返回Play对象"
// @Failure      400 {string} string "请求解码失败/参数非法/Slice不存在/Play已存在"
// @Failure      404 {string} string "关联Slice不存在"
// @Failure      500 {string} string "存储失败/部署失败/响应编码失败"
// @Router       /play [post]
func (s *Server) createPlay(w http.ResponseWriter, r *http.Request) {
	var play model.Play
	if err := json.NewDecoder(r.Body).Decode(&play); err != nil {
		http.Error(w, "请求解码失败", http.StatusBadRequest)
		return
	}

	// 检查值是否有效
	if err := play.Validate(); err != nil {
		slog.Error("非法值", "error", err)
		http.Error(w, fmt.Sprintf("非法值: %v", err), http.StatusBadRequest)
		return
	}

	// 检查slice是否存在
	if _, err := s.store.GetSliceBySliceID(play.SliceID); err != nil {
		if isNotFoundError(err) {
			slog.Warn("切片不存在", "sliceID", play.SliceID)
			http.Error(w, "切片不存在", http.StatusNotFound)
			return
		}
		slog.Error("获取切片失败", "sliceID", play.SliceID, "error", err)
		http.Error(w, "获取切片失败", http.StatusInternalServerError)
		return
	}

	// 检查是否有重复的play
	_, err := s.store.GetPlayBySliceID(play.SliceID)
	if err == nil {
		http.Error(w, "play已存在", http.StatusBadRequest)
		return
	}

	// 存储play
	play, err = s.store.CreatePlay(play)
	if err != nil {
		http.Error(w, "创建play失败", http.StatusInternalServerError)
		return
	}

	// 部署play
	err = s.kubeclient.Play(play, s.config.Namespace)
	if err != nil {
		// 删除存储
		errD := s.store.DeletePlay(play.ID.Hex())
		if errD != nil {
			slog.Error("删除play失败", "sliceID", play.SliceID, "err", errD)
		}
		http.Error(w, "部署play失败", http.StatusInternalServerError)
		return
	}

	// 返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(play); err != nil {
		slog.Error("响应编码失败", "error", err)
		http.Error(w, "响应编码失败", http.StatusInternalServerError)
		return
	}
	slog.Debug("创建play成功", "playID", play.ID.Hex())
}

// getPlay godoc
// @Summary      获取单个Play
// @Description  根据Play ID获取资源详情
// @Tags         Play
// @Accept       json
// @Produce      json
// @Param        playID path string true "Play ID"
// @Success      200 {object} model.Play "获取成功"
// @Failure      400 {string} string "缺少Play ID"
// @Failure      404 {string} string "Play不存在"
// @Failure      500 {string} string "获取失败/响应编码失败"
// @Router       /play/{play_id} [get]
func (s *Server) getPlay(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取play请求", "method", r.Method, "url", r.URL.String())
	playID := chi.URLParam(r, "play_id")
	if playID == "" {
		slog.Warn("缺少playID参数")
		http.Error(w, "缺少playID参数", http.StatusBadRequest)
		return
	}

	// 从对象存储中获取play对象
	play, err := s.store.GetPlay(playID)
	if err != nil {
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("play不存在", "playID", playID)
			http.Error(w, "play不存在", http.StatusNotFound)
			return
		}

		slog.Error("获取play失败", "playID", playID, "error", err)
		http.Error(w, "获取play失败", http.StatusInternalServerError)
		return
	}
	// 返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(play); err != nil {
		slog.Error("响应编码失败", "playID", playID, "sliceID", play.SliceID, "error", err)
		http.Error(w, "响应编码失败", http.StatusInternalServerError)
		return
	}
	slog.Debug("获取play成功", "playID", play.ID.Hex(), "sliceID", play.SliceID)
}

// updatePlay godoc
// @Summary      更新Play资源
// @Description  根据Play ID更新资源并重新部署
// @Tags         Play
// @Accept       json
// @Produce      json
// @Param        playID path string true "Play ID"
// @Param        play body model.Play true "更新后的Play对象"
// @Success      200 {object} model.Play "更新成功返回对象"
// @Failure      400 {string} string "缺少Play ID/请求解码失败/参数非法"
// @Failure      404 {string} string "Play不存在"
// @Failure      500 {string} string "更新失败/部署失败/响应编码失败"
// @Router       /play/{play_id} [put]
func (s *Server) updatePlay(w http.ResponseWriter, r *http.Request) {
	// slog.Debug("更新play请求", "method", r.Method, "url", r.URL.String())

	// 获取playID
	playID := chi.URLParam(r, "play_id")
	if playID == "" {
		slog.Warn("缺少playID参数")
		http.Error(w, "缺少playID参数", http.StatusBadRequest)
		return
	}

	// body中获取play更新参数
	// 动态更新, 若值为空则不更新
	var play model.Play
	if err := json.NewDecoder(r.Body).Decode(&play); err != nil {
		http.Error(w, "请求解码失败", http.StatusBadRequest)
		return
	}

	// 检查值是否有效
	if err := play.Validate(); err != nil {
		slog.Error("非法值", "error", err)
		http.Error(w, fmt.Sprintf("非法值: %v", err), http.StatusBadRequest)
		return
	}

	// 检查是否存在
	curPlay, err := s.store.GetPlay(play.ID.Hex())
	if err != nil {
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("play不存在", "playID", playID)
			http.Error(w, "play不存在", http.StatusNotFound)
			return
		}

		slog.Error("获取play失败", "playID", playID, "error", err)
		http.Error(w, "获取play失败", http.StatusInternalServerError)
		return
	}

	// 更新play
	err = curPlay.Update(play)
	if err != nil {
		slog.Error("更新play失败", "playID", playID, "error", err)
		http.Error(w, "更新play失败", http.StatusBadRequest)
		return
	}

	// 更新存储
	_, err = s.store.UpdatePlay(curPlay)
	if err != nil {
		slog.Error("更新play失败", "playID", playID, "error", err)
		http.Error(w, "更新play失败", http.StatusInternalServerError)
		return
	}

	// 更新部署
	err = s.kubeclient.Play(curPlay, s.config.Namespace)
	if err != nil {
		slog.Error("更新play部署失败", "playID", playID, "error", err)
		http.Error(w, "更新play部署失败", http.StatusInternalServerError)
		return
	}

	// 返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(curPlay); err != nil {
		slog.Error("响应编码失败", "playID", playID, "sliceID", curPlay.SliceID, "error", err)
		http.Error(w, "响应编码失败", http.StatusInternalServerError)
		return
	}
	slog.Debug("更新play成功", "playID", curPlay.ID.Hex(), "sliceID", curPlay.SliceID)
}
