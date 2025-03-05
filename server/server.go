package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/model"
	"slicer/render"
)

const SliceBucket = "slice"
const KubeBucket = "kube"

// Server holds the dependencies for the HTTP server.
type Server struct {
	store      db.Store
	router     *http.ServeMux
	render     render.Render
	kubeclient kubeclient.KubeClient
}

// NewServer creates a new HTTP server and sets up routing.
func NewServer(store db.Store, render render.Render, kubeclient kubeclient.KubeClient) *Server {
	s := &Server{
		store:      store,
		router:     http.NewServeMux(),
		render:     render,
		kubeclient: kubeclient,
	}
	s.routes()
	return s
}

// routes sets up the routes for the server.
func (s *Server) routes() {
	s.router.HandleFunc("POST /slice", s.createSlice)
	s.router.HandleFunc("DELETE /slice/{sst-sd}", s.deleteSlice)
	s.router.HandleFunc("GET /slice/{sst-sd}", s.getSlice)
}

type createSliceRequest struct {
	slice model.Slice
}

type createSliceResponse struct {
}

// createSlice handles the creation of a new slice.
func (s *Server) createSlice(w http.ResponseWriter, r *http.Request) {
	var slice model.Slice

	if err := json.NewDecoder(r.Body).Decode(&slice); err != nil {
		http.Error(w, "Failed to decode slice", http.StatusBadRequest)
		return
	}

	// 存储 slice对象
	filename := fmt.Sprintf("%d-%s.json", slice.SST, slice.SD)

	// Convert to k8s deployment file (not implemented)
	// s.render.Translate(slice)

	// Deploy to k8s (not implemented)
	// s.kubeclient.Deploy(slice)

	w.WriteHeader(http.StatusCreated)
}

// deleteSlice handles the deletion of a slice.
func (s *Server) deleteSlice(w http.ResponseWriter, r *http.Request) {
	filename := fmt.Sprintf("%s_%s.json", vars["sst"], vars["sd"])
	if err := os.Remove(filename); err != nil {
		http.Error(w, "Failed to delete slice", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// getSlice handles the retrieval of a slice.
func (s *Server) getSlice(w http.ResponseWriter, r *http.Request) {
	filename := fmt.Sprintf("%s_%s.json", vars["sst"], vars["sd"])
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		http.Error(w, "Failed to read slice", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

// Start starts the HTTP server.
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}
