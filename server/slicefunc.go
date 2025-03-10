package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"slicer/db"
	"slicer/model"
)

type createSliceRequest struct {
	Slice model.Slice `json:"slice"`
}

func (s *Server) createSlice(w http.ResponseWriter, r *http.Request) {
	var createSliceRequest createSliceRequest

	if err := json.NewDecoder(r.Body).Decode(&createSliceRequest); err != nil {
		http.Error(w, "Failed to decode request", http.StatusBadRequest)
		return
	}

	slice := createSliceRequest.Slice
	// 存储 slice对象
	err := s.setSlice(slice)
	if err != nil {
		http.Error(w, "Failed to store slice", http.StatusInternalServerError)
		return
	}

	// 切片转化为k8s资源
	dirPath, err := s.render.SliceToKube(slice)
	defer os.RemoveAll(dirPath) // 删除临时文件夹
	if err != nil {
		http.Error(w, "Failed to render slice", http.StatusInternalServerError)
		return
	}

	// 部署k8s资源
	err = s.kubeclient.ApplyDir(dirPath, s.config.Namespace)
	if err != nil {
		http.Error(w, "Failed to apply kube resources", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) deleteSlice(w http.ResponseWriter, r *http.Request) {
	sliceId := r.URL.Query().Get("sliceId")
	if sliceId == "" {
		http.Error(w, "sliceId is required", http.StatusBadRequest)
		return
	}

	// 从对象存储中获取slice对象
	slice, err := s.getSliceBySliceId(sliceId)
	if err != nil {
		http.Error(w, "Failed to get slice", http.StatusInternalServerError)
		return
	}

	// 切片转化为k8s资源
	dirPath, err := s.render.SliceToKube(slice)
	defer os.RemoveAll(dirPath) // 删除临时文件夹
	if err != nil {
		http.Error(w, "Failed to render slice", http.StatusInternalServerError)
		return
	}

	// 删除k8s资源
	err = s.kubeclient.DeleteDir(dirPath, s.config.Namespace)
	if err != nil {
		http.Error(w, "Failed to delete kube resources", http.StatusInternalServerError)
		return
	}

	// 删除对象存储中的slice对象
	if err := s.store.Delete(s.config.SliceStoreName, sliceId); err != nil {
		http.Error(w, "Failed to delete slice", http.StatusInternalServerError)
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
		http.Error(w, "sliceId is required", http.StatusBadRequest)
		return
	}

	// 从对象存储中获取slice对象
	slice, err := s.getSliceBySliceId(sliceId)
	if err != nil {
		http.Error(w, "Failed to get slice", http.StatusInternalServerError)
		return
	}

	getSliceResponse := getSliceResponse{
		Slice: slice,
	}

	//设置响应头
	w.Header().Set("Content-Type", "application/json")
	//编码响应
	if err := json.NewEncoder(w).Encode(getSliceResponse); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// 存储slice对象
func (s *Server) setSlice(slice model.Slice) error {
	sliceYAML, err := slice.ToYAML()
	if err != nil {
		return err
	}
	doc := &db.Document{
		ID:   slice.ID(),
		Data: sliceYAML,
	}
	return s.store.Insert(s.config.SliceStoreName, doc)
}

// 从存储中获取slice对象
func (s *Server) getSliceBySliceId(sliceId string) (model.Slice, error) {
	var slice model.Slice
	doc, err := s.store.Find(s.config.SliceStoreName, sliceId)
	if err != nil {
		return slice, err
	}
	data, ok := doc.Data.([]byte)
	if !ok {
		return slice, errors.New("failed to assert doc.Data to []byte")
	}
	err = slice.FromYAML(data)
	return slice, err
}
