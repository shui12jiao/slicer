package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/model"
	"slicer/render"
	"slicer/util"
)

// Server 负责处理HTTP请求
type Server struct {
	config     util.Config
	store      db.Store
	router     *http.ServeMux
	render     render.Render
	kubeclient kubeclient.KubeClient
}

func NewServer(config util.Config, store db.Store, render render.Render, kubeclient kubeclient.KubeClient) *Server {
	s := &Server{
		config:     config,
		store:      store,
		router:     http.NewServeMux(),
		render:     render,
		kubeclient: kubeclient,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.HandleFunc("POST /slice", s.createSlice)
	s.router.HandleFunc("DELETE /slice/{sliceId}", s.deleteSlice)
	s.router.HandleFunc("GET /slice/{sliceId}", s.getSlice)
}

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
	_, err := s.setSlice(slice)
	if err != nil {
		http.Error(w, "Failed to store slice", http.StatusInternalServerError)
		return
	}

	// 切片转化为k8s资源 TODO
	// s.render.Translate(slice)

	// 部署k8s资源 TODO
	// s.kubeclient.Deploy(slice)

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

	// 删除k8s资源 TODO
	// s.kubeclient.Delete(slice)

	// 删除对象存储中的slice对象
	if err := s.store.Delete(s.config.SliceBucket, sliceId); err != nil {
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
func (s *Server) setSlice(slice model.Slice) (string, error) {
	sliceYAML, err := slice.ToYAML()
	if err != nil {
		return "", err
	}
	filename := slice.ID() + ".yaml"
	if err := s.store.Upload(s.config.SliceBucket, filename, bytes.NewReader(sliceYAML)); err != nil {
		return "", err
	}

	return filename, nil
}

// 从对象存储中获取slice对象
func (s *Server) getSliceBySliceId(sliceId string) (model.Slice, error) {
	var slice model.Slice

	var buf bytes.Buffer
	if err := s.store.Download(s.config.SliceBucket, sliceId, &buf); err != nil {
		return slice, err
	}

	err := slice.FromYAML(buf.Bytes())
	return slice, err
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	return http.ListenAndServe(s.config.HTTPServerAddress, s.router)
}
