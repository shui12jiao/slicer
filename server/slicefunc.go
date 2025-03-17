package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slicer/model"
)

type createSliceRequest struct {
	Slice model.Slice `json:"slice"`
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
	err = s.storeSlice(wrappedSlice)
	if err != nil {
		http.Error(w, fmt.Sprintf("存储slice失败: %v", err), http.StatusInternalServerError)
		return
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if deleteErr := s.deleteSliceFromStore(slice.ID()); deleteErr != nil { //从mongodb中删除存储的slice文件
			// 记录删除 slice 时的错误，避免覆盖原始错误
			log.Printf("从存储中删除slice失败: %v", deleteErr)
		}
	})

	// 切片转化为k8s资源
	contents, err := s.render.SliceToKube(wrappedSlice)
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

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) deleteSlice(w http.ResponseWriter, r *http.Request) {
	sliceId := r.PathValue("sliceId")
	if sliceId == "" {
		http.Error(w, "缺少sliceId参数", http.StatusBadRequest)
		return
	}

	// 从对象存储中获取slice对象
	slice, err := s.findSlice(sliceId)
	if err != nil {
		http.Error(w, fmt.Sprintf("获取slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 切片转化为k8s资源
	contents, err := s.render.SliceToKube(slice)
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
	err = s.deleteSliceFromStore(slice.ID())
	if err != nil {
		http.Error(w, fmt.Sprintf("从存储中删除slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type getSliceResponse struct {
	Slice model.Slice `json:"slice"`
}

func (s *Server) getSlice(w http.ResponseWriter, r *http.Request) {
	sliceId := r.URL.Query().Get("sliceId")
	if sliceId == "" {
		http.Error(w, "缺少sliceId参数", http.StatusBadRequest)
		return
	}

	// 从对象存储中获取slice对象
	slice, err := s.findSlice(sliceId)
	if err != nil {
		http.Error(w, fmt.Sprintf("获取slice失败: %v", err), http.StatusInternalServerError)
		return
	}

	getSliceResponse := getSliceResponse{
		Slice: slice.Slice,
	}

	//设置响应头
	w.Header().Set("Content-Type", "application/json")
	//编码响应
	if err := json.NewEncoder(w).Encode(getSliceResponse); err != nil {
		http.Error(w, fmt.Sprintf("响应编码失败: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
