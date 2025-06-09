package service

import (
	"errors"
	"fmt"
	"log/slog"
	"slicer/model"

	"go.mongodb.org/mongo-driver/mongo"
)

// 给slice分配IP
func (s *Service) allocateIP(slice model.SliceProfile) (model.SliceProfile, error) {
	// SessionSubnets []Subnet
	// UPFN3Addr      string
	// UPFN4Addr      string
	// SMFN3Addr      string
	// SMFN4Addr      string

	if !slice.AddressValue.IsEmpty() {
		slog.Debug("Slice已分配IP，直接返回", "sliceID", slice.SliceID())
		return slice, nil
	}

	sessionSubnets := []model.Subnet{}
	for range slice.Sessions {
		sessionSubnet, err := s.IPAM.AllocateSessionSubnet()
		if err != nil {
			return slice, err
		}
		sessionSubnets = append(sessionSubnets, model.Subnet(sessionSubnet))
	}
	upfN3Addr, err := s.IPAM.AllocateN3Addr()
	if err != nil {
		return slice, fmt.Errorf("分配UPF N3地址失败: %w", err)
	}
	upfN4Addr, err := s.IPAM.AllocateN4Addr()
	if err != nil {
		return slice, fmt.Errorf("分配UPF N4地址失败: %w", err)
	}
	smfN3Addr, err := s.IPAM.AllocateN3Addr()
	if err != nil {
		return slice, fmt.Errorf("分配SMF N3地址失败: %w", err)
	}
	smfN4Addr, err := s.IPAM.AllocateN4Addr()
	if err != nil {
		return slice, fmt.Errorf("分配SMF N4地址失败: %w", err)
	}

	// 更新slice的AddressValue
	slice.AddressValue = model.AddressValue{
		SessionSubnets: sessionSubnets,
		UPFN3Addr:      upfN3Addr,
		UPFN4Addr:      upfN4Addr,
		SMFN3Addr:      smfN3Addr,
		SMFN4Addr:      smfN4Addr,
	}
	slog.Debug("分配IP成功", "sliceID", slice.SliceID(), "sessionSubnets", sessionSubnets,
		"UPFN3Addr", upfN3Addr, "UPFN4Addr", upfN4Addr,
		"SMFN3Addr", smfN3Addr, "SMFN4Addr", smfN4Addr)
	return slice, nil
}

// 释放slice已分配的IP
func (s *Service) releaseIP(slice model.SliceProfile) error {
	var errs []error
	err := s.IPAM.ReleaseN3Addr(slice.SMFN3Addr)
	if err != nil {
		errs = append(errs, fmt.Errorf("释放SMF N3地址失败: %w", err))
	}

	err = s.IPAM.ReleaseN3Addr(slice.UPFN3Addr)
	if err != nil {
		errs = append(errs, fmt.Errorf("释放UPF N3地址失败: %w", err))
	}

	err = s.IPAM.ReleaseN4Addr(slice.SMFN4Addr)
	if err != nil {
		errs = append(errs, fmt.Errorf("释放SMF N4地址失败: %w", err))
	}

	err = s.IPAM.ReleaseN4Addr(slice.UPFN4Addr)
	if err != nil {
		errs = append(errs, fmt.Errorf("释放UPF N4地址失败: %w", err))
	}

	for _, sessionSubnet := range slice.SessionSubnets {
		err = s.IPAM.ReleaseSessionSubnet(string(sessionSubnet))
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
