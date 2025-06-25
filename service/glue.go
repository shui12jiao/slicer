package service

import (
	"fmt"
	"log/slog"
	"slicer/model"
)

func (s *Service) MdeInstall(sliceID string) error {
	if sliceID == "" {
		// 如果为空则为现有所有slice安装MDE

		// 获取所有Slice
		slices, err := s.ListSlices()
		if err != nil {
			slog.Error("创建MDE失败，获取所有Slice失败", "error", err)
			return fmt.Errorf("创建MDE失败，获取所有Slice失败: %w", err)
		}

		// 遍历所有Slice并安装MDE
		activated := make([]string, 0)
		for _, slice := range slices {
			if slice.MonitorRef != nil {
				continue // 已经启用MDE的Slice跳过
			}

			// 更新Slice信息和部署
			if _, err = s.UpdateSlice(slice); err != nil {
				slog.Error("创建MDE失败, 更新Slice失败", "sliceID", slice.SliceID(), "error", err)
				return fmt.Errorf("更新Slice失败: %w", err)
			}
			// 添加到已激活列表
			activated = append(activated, slice.SliceID())
		}

		// 存储MDE信息
		if len(activated) > 0 {
			if err = s.Store.CreateMonitor(&model.Monitor{
				KPI: model.KPI{
					KPIName:        "slice_throughput",
					KPIDescription: "Slice Throughput",
					SubCounter: model.SubCounter{
						SubCounterType: "slice",
						SubCounterIDs:  activated,
					},
					Units: "Mbps",
				},
				RequestID: "foo",
			}); err != nil {
				slog.Error("创建MDE失败，存储MDE信息失败", "error", err)
				return fmt.Errorf("创建MDE失败，存储MDE信息失败: %w", err)
			}
		}
		slog.Info("MDE安装成功", "activatedSlices", activated)
		return nil
	} else {
		// 检查sliceID是否存在
		slice, err := s.GetSlice(sliceID)
		if err != nil {
			slog.Error("创建MDE失败，Slice不存在", "sliceID", sliceID, "error", err)
			return fmt.Errorf("创建MDE失败，Slice不存在: %w", err)
		}

		// 检查是否已启用MDE
		if slice.MonitorRef != nil {
			slog.Warn("创建MDE失败，Slice已启用MDE", "sliceID", sliceID)
			return fmt.Errorf("创建MDE失败，Slice已启用MDE")
		}

		// 更新Slice信息和部署
		if _, err = s.UpdateSlice(slice); err != nil {
			slog.Error("创建MDE失败, 更新Slice失败", "sliceID", sliceID, "error", err)
			return fmt.Errorf("更新Slice失败: %w", err)
		}

		// 存储MDE信息
		if err = s.Store.CreateMonitor(&model.Monitor{
			KPI: model.KPI{
				KPIName:        "slice_throughput",
				KPIDescription: "Slice Throughput",
				SubCounter: model.SubCounter{
					SubCounterType: "slice",
					SubCounterIDs:  []string{sliceID},
				},
				Units: "Mbps",
			},
			RequestID: "foo",
		}); err != nil {
			slog.Error("创建MDE失败，存储MDE信息失败", "sliceID", sliceID, "error", err)
			return fmt.Errorf("创建MDE失败，存储MDE信息失败: %w", err)
		}

		slog.Info("MDE安装成功", "sliceID", sliceID)
		return nil
	}
}

func (s *Service) MdeUninstall() error {
	monitors, err := s.Store.ListMonitor()
	if err != nil {
		slog.Error("卸载MDE失败，获取所有监控信息失败", "error", err)
		return fmt.Errorf("卸载MDE失败，获取所有监控信息失败: %w", err)
	}
	for _, monitor := range monitors {
		// 查找RequestID为"foo"的监控
		if monitor.RequestID == "foo" {
			sliceIDs := monitor.KPI.SubCounter.SubCounterIDs
			for _, sliceID := range sliceIDs {
				// 检查sliceID是否存在
				slice, err := s.GetSlice(sliceID)
				if err != nil {
					slog.Error("删除监控失败，Slice不存在", "sliceID", sliceID, "error", err)
					return fmt.Errorf("删除监控失败，Slice不存在: %w", err)
				}

				// 检查是否已启用监控
				if slice.MonitorRef == nil || *slice.MonitorRef != monitor.ID {
					slog.Warn("删除监控失败，Slice未启用监控或监控ID不匹配", "sliceID", sliceID, "monitorID", monitor.ID.Hex())
					return fmt.Errorf("删除监控失败，Slice未启用监控或监控ID不匹配")
				}

				// 更新Slice信息
				slice.MonitorRef = nil // 清除监控引用
				if _, err = s.UpdateSlice(slice); err != nil {
					slog.Error("删除监控失败, 更新Slice失败", "sliceID", sliceID, "error", err)
					return fmt.Errorf("更新Slice失败: %w", err)
				}
			}

			// 删除监控信息
			if err := s.Store.DeleteMonitor(monitor.ID.Hex()); err != nil {
				slog.Error("卸载MDE失败，删除监控信息失败", "monitorID", monitor.ID.Hex(), "error", err)
				return fmt.Errorf("卸载MDE失败，删除监控信息失败: %w", err)
			}
			slog.Info("MDE卸载成功", "monitorID", monitor.ID.Hex())
		}
	}
	return nil
}
