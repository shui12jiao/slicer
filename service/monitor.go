package service

import (
	"fmt"
	"log/slog"
	"slicer/model"
)

func (s *Service) CreateSliceMonitor(sliceID string, monitor *model.Monitor) (*model.Monitor, error) {
	// 检查sliceID是否存在
	slice, err := s.GetSlice(sliceID)
	if err != nil {
		slog.Error("创建监控失败，Slice不存在", "sliceID", sliceID, "error", err)
		return nil, fmt.Errorf("创建监控失败，Slice不存在: %w", err)
	}

	// 检查是否已启用监控
	if slice.MonitorRef != nil {
		slog.Warn("创建监控失败，Slice已启用监控", "sliceID", sliceID)
		return nil, fmt.Errorf("创建监控失败，Slice已启用监控")
	}

	// 存储监控信息
	if err = s.Store.CreateMonitor(monitor); err != nil {
		slog.Error("创建监控失败，存储监控信息失败", "sliceID", sliceID, "error", err)
		return nil, fmt.Errorf("创建监控失败，存储监控信息失败: %w", err)
	}

	// 更新Slice信息和部署
	slice.MonitorRef = &monitor.ID
	if _, err = s.UpdateSlice(slice); err != nil {
		slog.Error("创建监控失败, 更新Slice失败", "sliceID", sliceID, "error", err)
		return nil, fmt.Errorf("更新Slice失败: %w", err)
	}

	return monitor, nil
}

func (s *Service) DeleteSliceMonitor(sliceID, monitorID string) error {
	// 检查sliceID是否存在
	slice, err := s.GetSlice(sliceID)
	if err != nil {
		slog.Error("删除监控失败，Slice不存在", "sliceID", sliceID, "error", err)
		return fmt.Errorf("删除监控失败，Slice不存在: %w", err)
	}

	// 转换 monitorID 为 model.ObjectID
	monitorIDObj, err := ObjectIDFromString(monitorID)
	if err != nil {
		slog.Error("删除监控失败，监控ID格式错误", "monitorID", monitorID, "error", err)
		return fmt.Errorf("删除监控失败，监控ID格式错误: %w", err)
	}

	// 检查是否已启用监控
	if slice.MonitorRef == nil || *slice.MonitorRef != monitorIDObj {
		// 如果Slice未启用监控或监控ID不匹配，直接返回错误
		slog.Warn("删除监控失败，Slice未启用监控或监控ID不匹配", "sliceID", sliceID, "monitorID", monitorID)
		return fmt.Errorf("删除监控失败，Slice未启用监控或监控ID不匹配")
	}

	// 删除监控信息
	if err = s.Store.DeleteMonitor(monitorID); err != nil {
		slog.Error("删除监控失败，存储中删除监控信息失败", "monitorID", monitorID, "error", err)
		return fmt.Errorf("删除监控失败，存储中删除监控信息失败: %w", err)
	}

	// 更新Slice信息
	slice.MonitorRef = nil // 清除监控引用
	if _, err = s.UpdateSlice(slice); err != nil {
		slog.Error("删除监控失败, 更新Slice失败", "sliceID", sliceID, "error", err)
		return fmt.Errorf("更新Slice失败: %w", err)
	}

	return nil
}

func (s *Service) GetMonitor(monitorID string) (*model.Monitor, error) {
	// 从存储中获取监控信息
	monitor, err := s.Store.GetMonitor(monitorID)
	if err != nil {
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("监控不存在", "monitorID", monitorID)
			return nil, model.ErrMonitorNotFound
		}
		slog.Error("获取监控失败", "monitorID", monitorID, "error", err)
		return nil, fmt.Errorf("获取监控失败: %w", err)
	}
	return monitor, nil
}

func (s *Service) ListMonitor() ([]*model.Monitor, error) {
	// 从存储中获取所有监控信息
	monitors, err := s.Store.ListMonitor()
	if err != nil {
		slog.Error("获取监控列表失败", "error", err)
		return nil, fmt.Errorf("获取监控列表失败: %w", err)
	}
	return monitors, nil
}
