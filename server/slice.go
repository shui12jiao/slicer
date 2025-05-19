package server

import (
	"encoding/json"
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
// @Param        slice body model.Slice true "切片对象"
// @Success      200   {object}  model.SliceAndAddress "创建成功，返回切片及其地址"
// @Failure      400   {string}  string "请求格式错误或参数非法"
// @Failure      409   {string}  string "切片已存在"
// @Failure      500   {string}  string "服务器内部错误，如分配IP或部署资源失败"
// @Router       /slice [post]
func (s *Server) createSlice(w http.ResponseWriter, r *http.Request) {
	slog.Debug("创建slice请求", "method", r.Method, "url", r.URL.String())

	var slice model.Slice

	if err := json.NewDecoder(r.Body).Decode(&slice); err != nil {
		slog.Warn("请求解码失败", "error", err)
		http.Error(w, fmt.Sprintf("请求解码失败: %v", err), http.StatusBadRequest)
		return
	}

	// 检查值是否有效
	err := slice.Validate()
	if err != nil {
		slog.Warn("非法值", "error", err)
		http.Error(w, fmt.Sprintf("非法值: %v", err), http.StatusBadRequest)
		return
	}

	// 检查是否有重复的slice
	_, err = s.store.GetSliceBySliceID(slice.SliceID())
	if err == nil {
		slog.Warn("slice已存在", "sliceID", slice.SliceID())
		http.Error(w, fmt.Sprintf("slice已存在: %v", slice.SliceID()), http.StatusConflict)
		return
	}

	// 定义一个回滚栈，用于记录需要回滚的操作
	var rollbackFuncs []func()

	// 在函数退出时，根据是否出错决定是否执行回滚
	defer func() {
		if err != nil {
			slog.Debug("执行回滚操作")
			for i := len(rollbackFuncs) - 1; i >= 0; i-- {
				rollbackFuncs[i]()
			}
		}
	}()

	// 分配IP
	wrappedSlice, err := s.allocateIP(slice)
	if err != nil {
		slog.Error("分配IP失败", "error", err)
		http.Error(w, fmt.Sprintf("分配IP失败: %v", err), http.StatusInternalServerError)
		return
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if releaseErr := s.releaseIP(wrappedSlice); releaseErr != nil {
			slog.Error("回滚释放IP失败", "error", releaseErr)
		}
	})

	// 存储 slice对象
	wrappedSlice, err = s.store.CreateSlice(wrappedSlice)
	if err != nil {
		slog.Error("存储slice失败", "error", err)
		http.Error(w, fmt.Sprintf("存储slice失败: %v", err), http.StatusInternalServerError)
		return
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if deleteErr := s.store.DeleteSlice(wrappedSlice.ID.Hex()); deleteErr != nil {
			slog.Error("回滚存储中删除slice失败", "error", deleteErr)
		}
	})

	// 切片转化为k8s资源
	contents, err := s.render.RenderSlice(wrappedSlice)
	if err != nil {
		slog.Error("渲染slice失败", "error", err)
		http.Error(w, fmt.Sprintf("渲染slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 部署k8s资源
	err = s.kubeclient.ApplySlice(contents)
	if err != nil {
		slog.Error("应用kube资源失败", "error", err)
		http.Error(w, fmt.Sprintf("应用kube资源失败: %v", err), http.StatusInternalServerError)
		return
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if deleteErr := s.kubeclient.DeleteMulti(contents, s.config.Namespace); deleteErr != nil {
			slog.Error("回滚集群中删除配置失败", "error", deleteErr)
		}
	})

	//设置响应头
	w.Header().Set("Content-Type", "application/json")
	//编码响应
	if err := json.NewEncoder(w).Encode(wrappedSlice); err != nil {
		slog.Error("响应编码失败", "error", err)
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("创建slice成功", "sliceID", wrappedSlice.ID.Hex())
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

	// 从对象存储中获取slice对象
	slice, err := s.store.GetSliceBySliceID(sliceID)
	if err != nil {
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("slice不存在", "sliceID", sliceID)
			http.Error(w, fmt.Sprintf("slice不存在: %v", sliceID), http.StatusNotFound)
			return
		}

		slog.Error("获取slice失败", "sliceID", sliceID, "error", err)
		http.Error(w, fmt.Sprintf("获取slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 切片转化为k8s资源
	contents, err := s.render.RenderSlice(slice)
	if err != nil {
		slog.Error("渲染slice失败", "sliceID", sliceID, "error", err)
		http.Error(w, fmt.Sprintf("渲染slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 删除k8s资源
	err = s.kubeclient.DeleteSlice(contents)
	if err != nil {
		slog.Error("删除kube资源失败", "sliceID", sliceID, "error", err)
		http.Error(w, fmt.Sprintf("删除kube资源失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 释放IP
	err = s.releaseIP(slice)
	if err != nil {
		slog.Error("释放IP失败", "sliceID", sliceID, "error", err)
		http.Error(w, fmt.Sprintf("释放IP失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 删除对象存储中的slice对象
	err = s.store.DeleteSlice(slice.ID.Hex())
	if err != nil {
		slog.Error("从存储中删除slice失败", "sliceID", sliceID, "error", err)
		http.Error(w, fmt.Sprintf("从存储中删除slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("删除slice成功", "sliceID", sliceID)
	w.WriteHeader(http.StatusNoContent)
}

// getSlice godoc
// @Summary      获取单个切片
// @Description  根据切片ID获取指定切片的详细信息
// @Tags         Slice
// @Accept       json
// @Produce      json
// @Param        sliceID path string true "切片ID"
// @Success      200 {object} model.SliceAndAddress "获取成功"
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

	// 从对象存储中获取slice对象
	slice, err := s.store.GetSliceBySliceID(sliceID)
	if err != nil {
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("slice不存在", "sliceID", sliceID)
			http.Error(w, fmt.Sprintf("slice不存在: %v", sliceID), http.StatusNotFound)
			return
		}

		slog.Error("获取slice失败", "sliceID", sliceID, "error", err)
		http.Error(w, fmt.Sprintf("获取slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	//设置响应头
	w.Header().Set("Content-Type", "application/json")
	//编码响应
	if err := json.NewEncoder(w).Encode(slice); err != nil {
		slog.Error("响应编码失败", "sliceID", sliceID, "error", err)
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取slice成功", "sliceID", sliceID)
}

// listSlice godoc
// @Summary      获取所有切片
// @Description  获取当前系统中的所有切片列表
// @Tags         Slice
// @Accept       json
// @Produce      json
// @Success      200 {array} model.SliceAndAddress "获取成功，返回切片列表"
// @Failure      500 {string} string "服务器内部错误（获取列表失败、响应编码失败）"
// @Router       /slice [get]
func (s *Server) listSlice(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取slice列表请求", "method", r.Method, "url", r.URL.String())

	slices, err := s.store.ListSlice()
	if err != nil { // 为空时list不会返回错误
		// if isNotFoundError(err) { // MongoDB为空文档
		// 	slog.Debug("slice列表为空")
		// 	w.WriteHeader(http.StatusOK)
		// 	return
		// }

		slog.Error("获取slice列表失败", "error", err)
		http.Error(w, fmt.Sprintf("获取slice列表失败: %v", err), http.StatusInternalServerError)
		return
	}

	//设置响应头
	w.Header().Set("Content-Type", "application/json")
	//编码响应
	if err := json.NewEncoder(w).Encode(slices); err != nil {
		slog.Error("响应编码失败", "error", err)
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取slice列表成功", "count", len(slices))
}
