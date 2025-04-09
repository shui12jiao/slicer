package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slicer/model"
)

// createSlice 创建一个新的slice
// POST /api/slice
type createSliceRequest struct {
	Slice model.Slice `json:"slice"`
}

type createSliceResponse struct {
	Slice model.SliceAndAddress `json:"slice"`
}

func (s *Server) createSlice(w http.ResponseWriter, r *http.Request) {
	slog.Debug("创建slice请求", "method", r.Method, "url", r.URL.String())

	var createSliceRequest createSliceRequest

	if err := json.NewDecoder(r.Body).Decode(&createSliceRequest); err != nil {
		slog.Warn("请求解码失败", "error", err)
		http.Error(w, fmt.Sprintf("请求解码失败: %v", err), http.StatusBadRequest)
		return
	}

	slice := createSliceRequest.Slice

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
		http.Error(w, fmt.Sprintf("slice已存在: %v", slice.SliceID()), http.StatusBadRequest)
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
	err = s.kubeclient.ApplyMulti(contents, s.config.Namespace)
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
	if err := json.NewEncoder(w).Encode(createSliceResponse{
		Slice: wrappedSlice,
	}); err != nil {
		slog.Error("响应编码失败", "error", err)
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("创建slice成功", "sliceID", wrappedSlice.ID.Hex())
}

// deleteSlice 删除一个slice
// DELETE /api/slice/{sliceId}
func (s *Server) deleteSlice(w http.ResponseWriter, r *http.Request) {
	slog.Debug("删除slice请求", "method", r.Method, "url", r.URL.String())

	sliceId := r.PathValue("sliceId")
	if sliceId == "" {
		slog.Warn("缺少sliceId参数")
		http.Error(w, "缺少sliceId参数", http.StatusBadRequest)
		return
	}

	// 从对象存储中获取slice对象
	slice, err := s.store.GetSliceBySliceID(sliceId)
	if err != nil {
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("slice不存在", "sliceID", sliceId)
			http.Error(w, fmt.Sprintf("slice不存在: %v", sliceId), http.StatusNotFound)
			return
		}

		slog.Error("获取slice失败", "sliceID", sliceId, "error", err)
		http.Error(w, fmt.Sprintf("获取slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 切片转化为k8s资源
	contents, err := s.render.RenderSlice(slice)
	if err != nil {
		slog.Error("渲染slice失败", "sliceID", sliceId, "error", err)
		http.Error(w, fmt.Sprintf("渲染slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 删除k8s资源
	err = s.kubeclient.DeleteMulti(contents, s.config.Namespace)
	if err != nil {
		slog.Error("删除kube资源失败", "sliceID", sliceId, "error", err)
		http.Error(w, fmt.Sprintf("删除kube资源失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 释放IP
	err = s.releaseIP(slice)
	if err != nil {
		slog.Error("释放IP失败", "sliceID", sliceId, "error", err)
		http.Error(w, fmt.Sprintf("释放IP失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 删除对象存储中的slice对象
	err = s.store.DeleteSlice(slice.ID.Hex())
	if err != nil {
		slog.Error("从存储中删除slice失败", "sliceID", sliceId, "error", err)
		http.Error(w, fmt.Sprintf("从存储中删除slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("删除slice成功", "sliceID", sliceId)
	w.WriteHeader(http.StatusNoContent)
}

// getSlice 获取一个slice
// GET /api/slice/{sliceId}
type getSliceResponse struct {
	Slice model.SliceAndAddress `json:"slice"`
}

func (s *Server) getSlice(w http.ResponseWriter, r *http.Request) {
	slog.Debug("获取slice请求", "method", r.Method, "url", r.URL.String())

	sliceId := r.PathValue("sliceId")
	if sliceId == "" {
		slog.Warn("缺少sliceId参数")
		http.Error(w, "缺少sliceId参数", http.StatusBadRequest)
		return
	}

	// 从对象存储中获取slice对象
	slice, err := s.store.GetSliceBySliceID(sliceId)
	if err != nil {
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("slice不存在", "sliceID", sliceId)
			http.Error(w, fmt.Sprintf("slice不存在: %v", sliceId), http.StatusNotFound)
			return
		}

		slog.Error("获取slice失败", "sliceID", sliceId, "error", err)
		http.Error(w, fmt.Sprintf("获取slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	//设置响应头
	w.Header().Set("Content-Type", "application/json")
	//编码响应
	if err := json.NewEncoder(w).Encode(getSliceResponse{
		Slice: slice,
	}); err != nil {
		slog.Error("响应编码失败", "sliceID", sliceId, "error", err)
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取slice成功", "sliceID", sliceId)
}

// listSlice 获取所有slice
// GET /api/slice
type listSliceResponse struct {
	Slices []model.SliceAndAddress `json:"slices"`
}

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
	if err := json.NewEncoder(w).Encode(listSliceResponse{
		Slices: slices,
	}); err != nil {
		slog.Error("响应编码失败", "error", err)
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Debug("获取slice列表成功", "count", len(slices))
}
