package server

import (
	"errors"
	"fmt"
	"log/slog"
	"slicer/model"

	"go.mongodb.org/mongo-driver/mongo"
)

// 给slice分配IP
func (s *Server) allocateIP(slice model.Slice) (ws model.SliceAndAddress, err error) {
	// SessionSubnets []string
	// UPFN3Addr      string
	// UPFN4Addr      string
	// SMFN3Addr      string
	// SMFN4Addr      string
	sessionSubnets := []string{}
	for range slice.Sessions {
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

// 释放slice已分配的IP
func (s *Server) releaseIP(slice model.SliceAndAddress) error {
	var errs []error
	err := s.ipam.ReleaseN3Addr(slice.SMFN3Addr)
	if err != nil {
		errs = append(errs, fmt.Errorf("释放SMF N3地址失败: %w", err))
	}

	err = s.ipam.ReleaseN3Addr(slice.UPFN3Addr)
	if err != nil {
		errs = append(errs, fmt.Errorf("释放UPF N3地址失败: %w", err))
	}

	err = s.ipam.ReleaseN4Addr(slice.SMFN4Addr)
	if err != nil {
		errs = append(errs, fmt.Errorf("释放SMF N4地址失败: %w", err))
	}

	err = s.ipam.ReleaseN4Addr(slice.UPFN4Addr)
	if err != nil {
		errs = append(errs, fmt.Errorf("释放UPF N4地址失败: %w", err))
	}

	for _, sessionSubnet := range slice.SessionSubnets {
		err = s.ipam.ReleaseSessionSubnet(sessionSubnet)
		if err != nil {
			errs = append(errs, fmt.Errorf("释放会话子网%s失败: %w", sessionSubnet, err))
		}
	}
	return errors.Join(errs...)
}

func isNotFoundError(err error) bool {
	if errors.Is(err, mongo.ErrNoDocuments) { // MongoDB为空文档
		slog.Debug("MongoDB没有文档", "error", err)
		return true
	} else if errors.Is(err, mongo.ErrNilDocument) { // MongoDB没有文档
		slog.Debug("MongoDB返回空文档", "error", err)
		return true
	} else {
		slog.Debug("MongoDB返回错误", "error", err)
		return false
	}
}
