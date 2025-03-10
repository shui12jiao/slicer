package server

import (
	"encoding/json"
	"errors"
	"net/http"
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

	// 分配IP
	wrappedSlice, err := s.allocateIP(slice)
	if err != nil {
		http.Error(w, "Failed to allocate IP", http.StatusInternalServerError)
		return
	}

	// 存储 slice对象
	err = s.storeSlice(wrappedSlice)
	if err != nil {
		http.Error(w, "Failed to store slice", http.StatusInternalServerError)
		return
	}

	// 切片转化为k8s资源
	contents, err := s.render.SliceToKube(wrappedSlice)
	if err != nil {
		http.Error(w, "Failed to render slice", http.StatusInternalServerError)
		return
	}

	// 部署k8s资源
	err = s.kubeclient.ApplyMulti(contents, s.config.Namespace)
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
	contents, err := s.render.SliceToKube(slice)
	if err != nil {
		http.Error(w, "Failed to render slice", http.StatusInternalServerError)
		return
	}

	// 删除k8s资源
	err = s.kubeclient.DeleteMulti(contents, s.config.Namespace)
	if err != nil {
		http.Error(w, "Failed to delete kube resources", http.StatusInternalServerError)
		return
	}

	// 释放IP
	err = s.ipam.ReleaseSessionSubnets(slice.SessionSubnets) // 释放各个session的子网
	if err != nil {
		http.Error(w, "Failed to release session subnets", http.StatusInternalServerError)
		return
	}
	err = s.ipam.ReleaseN3Addr(slice.UPFN3Addr) // 释放UPF N3地址
	if err != nil {
		http.Error(w, "Failed to release UPF N3 address", http.StatusInternalServerError)
	}
	err = s.ipam.ReleaseN4Addr(slice.UPFN4Addr) // 释放UPF N4地址
	if err != nil {
		http.Error(w, "Failed to release UPF N4 address", http.StatusInternalServerError)
	}
	err = s.ipam.ReleaseN3Addr(slice.SMFN3Addr) // 释放SMF N3地址
	if err != nil {
		http.Error(w, "Failed to release SMF N3 address", http.StatusInternalServerError)
	}
	err = s.ipam.ReleaseN4Addr(slice.SMFN4Addr) // 释放SMF N4地址
	if err != nil {
		http.Error(w, "Failed to release SMF N4 address", http.StatusInternalServerError)
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
		Slice: slice.Slice,
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
func (s *Server) storeSlice(slice model.SliceAndAddress) error {
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
func (s *Server) getSliceBySliceId(sliceId string) (slice model.SliceAndAddress, err error) {
	doc, err := s.store.Find(s.config.SliceStoreName, sliceId)
	if err != nil {
		return slice, err
	}
	data, ok := doc.Data.([]byte)
	if !ok {
		return slice, errors.New("failed to assert doc.Data to []byte")
	}
	err = slice.FromYAML(data)
	return
}

// 给slice分配IP
func (s *Server) allocateIP(slice model.Slice) (ws model.SliceAndAddress, err error) {
	// SessionSubnets []string
	// UPFN3Addr      string
	// UPFN4Addr      string
	// SMFN3Addr      string
	// SMFN4Addr      string
	sessionSubnets := []string{}
	for len(slice.Sessions) > 0 {
		sessionSubnet, err := s.ipam.AllocateSessionSubnet()
		if err != nil {
			return ws, err
		}
		sessionSubnets = append(sessionSubnets, sessionSubnet)
	}
	upfN3Addr, err := s.ipam.AllocateN3Addr()
	if err != nil {
		return
	}
	upfN4Addr, err := s.ipam.AllocateN4Addr()
	if err != nil {
		return
	}
	smfN3Addr, err := s.ipam.AllocateN3Addr()
	if err != nil {
		return
	}
	smfN4Addr, err := s.ipam.AllocateN4Addr()
	if err != nil {
		return
	}

	return model.SliceAndAddress{
		Slice: slice,
		AddressValue: model.AddressValue{
			SessionSubnets: sessionSubnets,
			UPFN3Addr:      upfN3Addr,
			UPFN4Addr:      upfN4Addr,
			SMFN3Addr:      smfN3Addr,
			SMFN4Addr:      smfN4Addr,
		},
	}, nil
}
