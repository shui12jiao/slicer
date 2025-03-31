package server

import (
	"encoding/json"
	"fmt"
	"log"
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
	var createSliceRequest createSliceRequest

	if err := json.NewDecoder(r.Body).Decode(&createSliceRequest); err != nil {
		http.Error(w, fmt.Sprintf("请求解码失败: %v", err), http.StatusBadRequest)
		return
	}

	slice := createSliceRequest.Slice

	// 检查值是否有效
	err := slice.Validate()
	if err != nil {
		http.Error(w, fmt.Sprintf("非法值: %v", err), http.StatusInternalServerError)
		return
	}

	// 检查是否有重复的slice
	_, err = s.store.GetSliceBySliceID(slice.SliceID())
	if err == nil {
		http.Error(w, fmt.Sprintf("slice已存在: %v", slice.SliceID()), http.StatusBadRequest)
	}

	// 定义一个回滚栈，用于记录需要回滚的操作
	var rollbackFuncs []func()

	// 在函数退出时，根据是否出错决定是否执行回滚
	defer func() {
		if err != nil {
			for i := len(rollbackFuncs) - 1; i >= 0; i-- {
				rollbackFuncs[i]()
			}
		}
	}()

	// 分配IP
	wrappedSlice, err := s.allocateIP(slice)
	if err != nil {
		http.Error(w, fmt.Sprintf("分配IP失败: %v", err), http.StatusInternalServerError)
		return
	}
	rollbackFuncs = append(rollbackFuncs, func() { //出错则释放IP
		if releaseErr := s.releaseIP(wrappedSlice); releaseErr != nil {
			// 记录释放IP时的错误，避免覆盖原始错误
			log.Printf("释放IP失败: %v", releaseErr)
		}
	})

	// 存储 slice对象
	wrappedSlice, err = s.store.CreateSlice(wrappedSlice) //返回的slice包含了ID
	if err != nil {
		http.Error(w, fmt.Sprintf("存储slice失败: %v", err), http.StatusInternalServerError)
		return
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if deleteErr := s.store.DeleteSlice(wrappedSlice.ID.Hex()); deleteErr != nil { //从mongodb中删除存储的slice文件
			// 记录删除 slice 时的错误，避免覆盖原始错误
			log.Printf("从存储中删除slice失败: %v", deleteErr)
		}
	})

	// 切片转化为k8s资源
	contents, err := s.render.RenderSlice(wrappedSlice)
	if err != nil {
		http.Error(w, fmt.Sprintf("渲染slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 部署k8s资源
	err = s.kubeclient.ApplyMulti(contents, s.config.Namespace)
	if err != nil {
		http.Error(w, fmt.Sprintf("应用kube资源失败: %v", err), http.StatusInternalServerError)
		return
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if deleteErr := s.kubeclient.DeleteMulti(contents, s.config.Namespace); deleteErr != nil { //保证原子操作
			log.Printf("从集群中删除配置失败: %v", deleteErr)
		}
	})

	//设置响应头
	w.Header().Set("Content-Type", "application/json")
	//编码响应
	// w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(createSliceResponse{
		Slice: wrappedSlice,
	}); err != nil {
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}
}

// deleteSlice 删除一个slice
// DELETE /api/slice/{sliceId}
func (s *Server) deleteSlice(w http.ResponseWriter, r *http.Request) {
	sliceId := r.PathValue("sliceId")
	if sliceId == "" {
		http.Error(w, "缺少sliceId参数", http.StatusBadRequest)
		return
	}

	// 从对象存储中获取slice对象
	slice, err := s.store.GetSliceBySliceID(sliceId)
	if err != nil {
		http.Error(w, fmt.Sprintf("获取slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 切片转化为k8s资源
	contents, err := s.render.RenderSlice(slice)
	if err != nil {
		http.Error(w, fmt.Sprintf("渲染slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 删除k8s资源
	err = s.kubeclient.DeleteMulti(contents, s.config.Namespace)
	if err != nil {
		http.Error(w, fmt.Sprintf("删除kube资源失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 释放IP
	err = s.releaseIP(slice)
	if err != nil {
		http.Error(w, fmt.Sprintf("释放IP失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 删除对象存储中的slice对象
	err = s.store.DeleteSlice(slice.ID.Hex())
	if err != nil {
		http.Error(w, fmt.Sprintf("从存储中删除slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getSlice 获取一个slice
// GET /api/slice/{sliceId}
type getSliceResponse struct {
	Slice model.SliceAndAddress `json:"slice"`
}

func (s *Server) getSlice(w http.ResponseWriter, r *http.Request) {
	sliceId := r.PathValue("sliceId")
	if sliceId == "" {
		http.Error(w, "缺少sliceId参数", http.StatusBadRequest)
		return
	}

	// 从对象存储中获取slice对象
	slice, err := s.store.GetSliceBySliceID(sliceId)
	if err != nil {
		http.Error(w, fmt.Sprintf("获取slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	//设置响应头
	w.Header().Set("Content-Type", "application/json")
	//编码响应
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(getSliceResponse{
		Slice: slice,
	}); err != nil {
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}
}

// listSlice 获取所有slice
// GET /api/slice
type listSliceResponse struct {
	Slices []model.SliceAndAddress `json:"slices"`
}

func (s *Server) listSlice(w http.ResponseWriter, r *http.Request) {
	slices, err := s.store.ListSlice()
	if err != nil {
		http.Error(w, fmt.Sprintf("获取slice列表失败: %v", err), http.StatusInternalServerError)
		return
	}

	//设置响应头
	w.Header().Set("Content-Type", "application/json")
	//编码响应
	if err := json.NewEncoder(w).Encode(listSliceResponse{
		Slices: slices,
	}); err != nil {
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}
}
