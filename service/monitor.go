package service

import (
	"fmt"
	"log/slog"
	"slicer/model"
)

func (s *Service) CreateSliceMonitor(sliceID string, monitor model.Monitor) (model.Monitor, error) {
	// 检查sliceID是否存在
	_, err := s.GetSlice(sliceID)
	if err != nil {
		slog.Error("创建监控失败，Slice不存在", "sliceID", sliceID, "error", err)
		return fmt.Errorf("创建监控失败，Slice不存在: %w", err)
	}

}
