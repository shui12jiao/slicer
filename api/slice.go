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

// createSlice godoc
// @Summary      创建切片
// @Description  接受一个切片对象，创建一个新的切片，并返回创建的切片对象
// @Tags         Slice
// @Accept       json
// @Produce      json
// @Param        slice body model.SliceProfile true "切片对象，包含切片ID、KubeConfig和SLA等信息"
// @Success      200   {object}  model.SliceProfile "创建成功，返回切片及其地址"
// @Failure      400   {string}  string "请求格式错误或参数非法"
// @Failure      409   {string}  string "切片已存在"
// @Failure      500   {string}  string "服务器内部错误，如分配IP或部署资源失败"
// @Router       /slice [post]
func (s *Server) createSlice(w http.ResponseWriter, r *http.Request) {
	slog.Debug("创建slice请求", "method", r.Method, "url", r.URL.String())

	var slice model.SliceProfile

	if err := json.NewDecoder(r.Body).Decode(&slice); err != nil {
		slog.Warn("请求解码失败", "error", err)
		http.Error(w, fmt.Sprintf("请求解码失败: %v", err), http.StatusBadRequest)
		return
	}

	// 检查值是否有效
	err := slice.Slice.Validate()
	if err != nil {
		slog.Warn("非法值", "error", err)
		http.Error(w, fmt.Sprintf("非法值: %v", err), http.StatusBadRequest)
		return
	}

	if slice, err = s.service.CreateSlice(slice); err != nil {
		slog.Error("创建slice失败", "error", err)
		if err == model.ErrSliceAlreadyExists {
			http.Error(w, fmt.Sprintf("切片已存在: %v", slice.SliceID()), http.StatusConflict)
			return
		}
		http.Error(w, fmt.Sprintf("创建slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("创建slice请求成功", "sliceID", slice.SliceID())
	encodeResponse(w, slice)
}

// updateSlice godoc
// @Summary      更新切片
// @Description  接受一个切片对象，更新指定的切片，并返回更新后的切片对象
// @Tags         Slice
// @Accept       json
// @Produce      json
// @Param        slice body model.SliceProfile true "切片对象，包含切片ID、KubeConfig和SLA等信息"
// @Success      200 {object} model.SliceProfile "更新成功，返回更新后的切片对象"
// @Failure      400 {string} string "请求格式错误或参数非法"
// @Failure      404 {string} string "切片不存在"
// @Failure      500 {string} string "服务器内部错误（更新k8s资源失败、分配IP失败、存储更新失败）"
// @Router       /slice [put]
func (s *Server) updateSlice(w http.ResponseWriter, r *http.Request) {
	slog.Debug("更新slice请求", "method", r.Method, "url", r.URL.String())

	var slice model.SliceProfile

	if err := json.NewDecoder(r.Body).Decode(&slice); err != nil {
		slog.Warn("请求解码失败", "error", err)
		http.Error(w, fmt.Sprintf("请求解码失败: %v", err), http.StatusBadRequest)
		return
	}

	// 检查值是否有效
	err := slice.Slice.Validate()
	if err != nil {
		slog.Warn("非法值", "error", err)
		http.Error(w, fmt.Sprintf("非法值: %v", err), http.StatusBadRequest)
		return
	}

	slice, err = s.service.UpdateSlice(slice)
	if err != nil {
		slog.Error("更新slice失败", "error", err)
		if errors.Is(err, model.ErrSliceNotFound) {
			http.Error(w, fmt.Sprintf("切片不存在: %v", slice.SliceID()), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("更新slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("更新slice请求成功", "sliceID", slice.SliceID())
	encodeResponse(w, slice)

}

// deleteSlice godoc
// @Summary      删除切片
// @Description  根据切片ID删除指定的切片资源
// @Tags         Slice
// @Accept       json
// @Produce      json
// @Param        sliceID path string true "切片ID"
// @Success      204 "删除成功无内容"
// @Failure      400 {string} string "缺少sliceID参数"
// @Failure      404 {string} string "切片不存在"
// @Failure      500 {string} string "服务器内部错误（获取/渲染/删除k8s资源失败、释放IP失败、存储删除失败）"
// @Router       /slice/{slice_id} [delete]
func (s *Server) deleteSlice(w http.ResponseWriter, r *http.Request) {
	slog.Debug("删除slice请求", "method", r.Method, "url", r.URL.String())

	sliceID := chi.URLParam(r, "slice_id")
	if sliceID == "" {
		slog.Warn("缺少sliceID参数")
		http.Error(w, "缺少sliceID参数", http.StatusBadRequest)
		return
	}

	err := s.service.DeleteSlice(sliceID)
	if err != nil {
		if errors.Is(err, model.ErrSliceNotFound) {
			slog.Warn("slice不存在", "sliceID", sliceID)
			http.Error(w, fmt.Sprintf("slice不存在: %v", sliceID), http.StatusNotFound)
			return
		}
		slog.Error("删除slice请求失败", "sliceID", sliceID, "error", err)
		http.Error(w, fmt.Sprintf("删除slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("删除slice请求成功", "sliceID", sliceID)
	w.WriteHeader(http.StatusNoContent)
}

// getSlice godoc
// @Summary      获取单个切片
// @Description  根据切片ID获取指定切片的详细信息
// @Tags         Slice
// @Accept       json
// @Produce      json
// @Param        sliceID path string true "切片ID"
// @Success      200 {object} model.SliceProfile "获取成功"
// @Failure      400 {string} string "缺少sliceID参数"
// @Failure      404 {string} string "切片不存在"
// @Failure      500 {string} string "服务器内部错误（获取失败、响应编码失败）"
// @Router       /slice/{slice_id} [get]
func (s *Server) getSlice(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取slice请求", "method", r.Method, "url", r.URL.String())

	sliceID := chi.URLParam(r, "slice_id")
	if sliceID == "" {
		slog.Warn("缺少sliceID参数")
		http.Error(w, "缺少sliceID参数", http.StatusBadRequest)
		return
	}

	slice, err := s.service.GetSlice(sliceID)
	if err != nil {
		slog.Error("获取slice请求失败", "sliceID", sliceID, "error", err)
		// 如果是MongoDB的空文档错误，返回404
		if errors.Is(err, model.ErrSliceNotFound) {
			http.Error(w, fmt.Sprintf("切片不存在: %v", sliceID), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("获取slice失败: %v", err), http.StatusInternalServerError)
		}
		return
	}

	slog.Debug("获取slice成功", "sliceID", sliceID)
	encodeResponse(w, slice)
}

// listSlice godoc
// @Summary      获取所有切片
// @Description  获取当前系统中的所有切片列表
// @Tags         Slice
// @Accept       json
// @Produce      json
// @Success      200 {array} model.SliceProfile "获取成功，返回切片列表"
// @Failure      500 {string} string "服务器内部错误（获取列表失败、响应编码失败）"
// @Router       /slice [get]
func (s *Server) listSlice(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取slice列表请求", "method", r.Method, "url", r.URL.String())

	slices, err := s.service.ListSlices()
	if err != nil {
		slog.Error("获取slice列表请求失败", "error", err)
		http.Error(w, fmt.Sprintf("获取slice列表失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取slice列表成功", "count", len(slices))
	encodeResponse(w, slices)
}
