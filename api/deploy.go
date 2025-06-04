package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slicer/model"

	"github.com/go-chi/chi"
)

// updateDeploy godoc
// @Summary      更新Deploy资源
// @Description  根据Deploy ID更新资源并重新部署
// @Tags         Deploy
// @Accept       json
// @Produce      json
// @Param        deployID path string true "Deploy ID"
// @Param        deploy body model.Deploy true "更新后的Deploy对象"
// @Success      200 {object} model.Deploy "更新成功返回对象"
// @Failure      400 {string} string "缺少Deploy ID/请求解码失败/参数非法"
// @Failure      404 {string} string "Deploy不存在"
// @Failure      500 {string} string "更新失败/部署失败/响应编码失败"
// @Router       /deploy/{deploy_id} [put]
func (s *Server) updateDeploy(w http.ResponseWriter, r *http.Request) {
	// slog.Debug("更新deploy请求", "method", r.Method, "url", r.URL.String())

	// 获取deployID
	deployID := chi.URLParam(r, "deploy_id")
	if deployID == "" {
		slog.Warn("缺少deployID参数")
		http.Error(w, "缺少deployID参数", http.StatusBadRequest)
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

	// 检查是否存在
	curDeploy, err := s.store.GetDeploy(deploy.ID.Hex())
	if err != nil {
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("deploy不存在", "deployID", deployID)
			http.Error(w, "deploy不存在", http.StatusNotFound)
			return
		}

		slog.Error("获取deploy失败", "deployID", deployID, "error", err)
		http.Error(w, "获取deploy失败", http.StatusInternalServerError)
		return
	}

	// 更新deploy
	err = curDeploy.Update(deploy)
	if err != nil {
		slog.Error("更新deploy失败", "deployID", deployID, "error", err)
		http.Error(w, "更新deploy失败", http.StatusBadRequest)
		return
	}

	// 更新存储
	_, err = s.store.UpdateDeploy(curDeploy)
	if err != nil {
		slog.Error("更新deploy失败", "deployID", deployID, "error", err)
		http.Error(w, "更新deploy失败", http.StatusInternalServerError)
		return
	}

	// 更新部署
	err = s.kubeClient.Deploy(curDeploy, s.config.Namespace)
	if err != nil {
		slog.Error("更新deploy部署失败", "deployID", deployID, "error", err)
		http.Error(w, "更新deploy部署失败", http.StatusInternalServerError)
		return
	}

	// 返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(curDeploy); err != nil {
		slog.Error("响应编码失败", "deployID", deployID, "sliceID", curDeploy.SliceID, "error", err)
		http.Error(w, "响应编码失败", http.StatusInternalServerError)
		return
	}
	slog.Debug("更新deploy成功", "deployID", curDeploy.ID.Hex(), "sliceID", curDeploy.SliceID)
}
