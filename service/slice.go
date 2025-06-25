package service

import (
	"fmt"
	"log/slog"
	"slicer/model"
)

func (s *Service) CreateSlice(slice *model.SliceProfile) (*model.SliceProfile, error) {
	// 检查是否已存在同名的Slice
	_, err := s.GetSlice(slice.SliceID())
	if err == nil {
		// 如果获取Slice时没有错误，说明Slice已存在
		slog.Warn("创建Slice失败，Slice已存在", "sliceID", slice.SliceID())
		return slice, model.ErrSliceAlreadyExists
	} else if err != model.ErrSliceNotFound { // 如果是其他错误，返回错误
		slog.Error("检查同名Slice失败", "sliceID", slice.SliceID(), "error", err)
		return slice, fmt.Errorf("检查同名Slice失败: %w", err)
	}

	// 定义一个回滚栈，用于记录需要回滚的操作
	var rollbackFuncs []func()

	// 在函数退出时，根据是否出错决定是否执行回滚
	defer func() {
		if r := recover(); r != nil || err != nil {
			slog.Debug("执行回滚操作")
			for i := len(rollbackFuncs) - 1; i >= 0; i-- {
				rollbackFuncs[i]()
			}
			if r != nil {
				panic(r)
			}
		}
	}()

	// 分配IP地址
	if err = s.allocateIP(slice); err != nil {
		slog.Error("分配IP失败", "error", err)
		return slice, fmt.Errorf("分配IP失败: %w", err)
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if releaseErr := s.releaseIP(slice); releaseErr != nil {
			slog.Error("回滚释放IP失败", "error", releaseErr)
		}
	})

	// 存储 slice对象
	err = s.Store.CreateSlice(slice)
	if err != nil {
		slog.Error("存储slice失败", "error", err)
		return slice, fmt.Errorf("存储slice失败: %w", err)
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if deleteErr := s.Store.DeleteSlice(slice.ID.Hex()); deleteErr != nil {
			slog.Error("回滚: 删除slice失败", "error", deleteErr)
		}
	})

	// 切片转化为helm values
	sliceVals, commonVal, err := s.Open5gs.GenerateValues(slice.SliceID()) // 获取所有切片的Values
	if err != nil {
		slog.Error("生成Open5GS的Values失败", "error", err)
		return slice, fmt.Errorf("生成Open5GS的Values失败: %w", err)
	}

	// 部署slice的Helm Chart
	_, err = s.HelmClient.Install(
		s.Open5gs.HelmReleasePrefix+slice.SliceID(),
		s.Open5gs.HelmSliceChart,
		sliceVals[slice.SliceID()].ToMap(),
	)
	if err != nil {
		slog.Error("部署slice的Helm Chart失败", "error", err)
		return slice, fmt.Errorf("部署slice的Helm Chart失败: %w", err)
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if uninstallErr := s.HelmClient.Uninstall(s.Open5gs.HelmReleasePrefix + slice.SliceID()); uninstallErr != nil {
			slog.Error("回滚: 卸载slice的Helm Chart失败", "error", uninstallErr)
		}
	})
	// 部署common的Helm Chart
	_, err = s.HelmClient.Install(
		s.Open5gs.HelmReleasePrefix+"common",
		s.Open5gs.HelmCommonChart,
		commonVal.ToMap(),
	)
	if err != nil {
		slog.Error("部署common的Helm Chart失败", "error", err)
		return slice, fmt.Errorf("部署common的Helm Chart失败: %w", err)
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if uninstallErr := s.HelmClient.Uninstall(s.Open5gs.HelmReleasePrefix + "common"); uninstallErr != nil {
			slog.Error("回滚: 卸载common的Helm Chart失败", "error", uninstallErr)
		}
	})

	// 返回创建的切片信息
	slog.Info("创建Slice成功", "sliceID", slice.SliceID())
	return slice, nil
}

func (s *Service) UpdateSlice(slice *model.SliceProfile) (*model.SliceProfile, error) {
	// 检查slice是否存在, 并获取旧的slice对象
	sliceOld, err := s.GetSlice(slice.SliceID())
	if err != nil {
		return slice, err
	}

	// 定义一个回滚栈，用于记录需要回滚的操作
	var rollbackFuncs []func()

	// 在函数退出时，根据是否出错决定是否执行回滚
	defer func() {
		if r := recover(); r != nil || err != nil {
			slog.Debug("执行回滚操作")
			for i := len(rollbackFuncs) - 1; i >= 0; i-- {
				rollbackFuncs[i]()
			}
			if r != nil {
				panic(r)
			}
		}
	}()

	// 更新 slice对象
	err = s.Store.UpdateSlice(slice)
	if err != nil {
		slog.Error("更新slice失败", "error", err)
		return slice, fmt.Errorf("更新slice失败: %w", err)
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if updateErr := s.Store.UpdateSlice(sliceOld); updateErr != nil {
			slog.Error("回滚: 更新回旧slice失败", "error", updateErr)
		}
	})

	// 切片转化为helm values
	sliceVals, commonVal, err := s.Open5gs.GenerateValues(slice.SliceID()) // 获取所有切片的Values
	if err != nil {
		slog.Error("生成Open5GS的Values失败", "error", err)
		return slice, fmt.Errorf("生成Open5GS的Values失败: %w", err)
	}

	// 更新slice的Helm Chart
	sliceReleaseName := s.Open5gs.HelmReleasePrefix + slice.SliceID()
	sliceReleaseOld, err := s.HelmClient.Get(sliceReleaseName)
	_, err = s.HelmClient.Upgrade(
		sliceReleaseName,
		s.Open5gs.HelmSliceChart,
		sliceVals[slice.SliceID()].ToMap(),
	)
	if err != nil {
		slog.Error("部署slice的Helm Chart失败", "error", err)
		return slice, fmt.Errorf("部署slice的Helm Chart失败: %w", err)
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if rollbackErr := s.HelmClient.Rollback(sliceReleaseName, sliceReleaseOld.Version); rollbackErr != nil {
			slog.Error("回滚: 回滚slice的Helm Chart失败", "error", rollbackErr, "releaseName", sliceReleaseName, "version", sliceReleaseOld.Version)
		}
	})
	// 更新common的Helm Chart
	commonReleaseName := s.Open5gs.HelmReleasePrefix + "common"
	commonReleaseOld, err := s.HelmClient.Get(commonReleaseName)
	_, err = s.HelmClient.Upgrade(
		commonReleaseName,
		s.Open5gs.HelmCommonChart,
		commonVal.ToMap(),
	)
	if err != nil {
		slog.Error("部署common的Helm Chart失败", "error", err)
		return slice, fmt.Errorf("部署common的Helm Chart失败: %w", err)
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if rollbackErr := s.HelmClient.Rollback(commonReleaseName, commonReleaseOld.Version); rollbackErr != nil {
			slog.Error("回滚: 回滚common的Helm Chart失败", "error", rollbackErr, "releaseName", commonReleaseName, "version", commonReleaseOld.Version)
		}
	})

	// 返回更新的切片信息
	slog.Info("更新Slice成功", "sliceID", slice.SliceID())
	return slice, nil
}

func (s *Service) DeleteSlice(sliceID string) error {
	// 获取slice对象
	slice, err := s.GetSlice(sliceID)
	if err != nil {
		return err
	}

	// 首先删除监控
	if slice.MonitorRef != nil {
		err := s.Store.DeleteMonitor(slice.MonitorRef.Hex())
		if err != nil {
			slog.Error("从存储中删除监控失败", "sliceID", sliceID, "monitorID", slice.MonitorRef.Hex(), "error", err)
			return fmt.Errorf("从存储中删除监控失败: %w", err)
		}
		slog.Info("从存储中删除监控成功", "sliceID", sliceID, "monitorID", slice.MonitorRef.Hex())
		slice.MonitorRef = nil // 清除监控引用
	}

	// 从存储中删除slice对象
	if err := s.Store.DeleteSlice(slice.ID.Hex()); err != nil {
		slog.Error("从存储中删除slice失败", "sliceID", sliceID, "error", err)
		return fmt.Errorf("从存储中删除slice失败: %w", err)
	}

	// 释放slice已分配的IP地址
	if err = s.releaseIP(slice); err != nil {
		slog.Error("释放IP失败", "sliceID", sliceID, "error", err)
		return fmt.Errorf("释放IP失败: %w", err)
	}

	// 生成Values
	_, commonVal, err := s.Open5gs.GenerateValues(sliceID) // 仅获取公共值
	if err != nil {
		slog.Error("生成Open5GS的Values失败", "error", err)
		return fmt.Errorf("生成Open5GS的Values失败: %w", err)
	}

	// 卸载slice的Helm Chart
	if err := s.HelmClient.Uninstall(s.Open5gs.HelmReleasePrefix + slice.SliceID()); err != nil {
		slog.Error("卸载slice的Helm Chart失败", "sliceID", sliceID, "error", err)
		return fmt.Errorf("卸载slice的Helm Chart失败: %w", err)
	}
	// 更新common的Helm Chart
	if _, err := s.HelmClient.InstallOrUpgrade(
		s.Open5gs.HelmReleasePrefix+"common",
		s.Open5gs.HelmCommonChart,
		commonVal.ToMap(),
	); err != nil {
		slog.Error("更新common的Helm Chart失败", "error", err)
		return fmt.Errorf("更新common的Helm Chart失败: %w", err)
	}

	return nil
}

func (s *Service) GetSlice(sliceID string) (*model.SliceProfile, error) {
	// 从对象存储中获取slice对象
	slice, err := s.Store.GetSliceBySliceID(sliceID)
	if err != nil {
		if isNotFoundError(err) { // MongoDB为空文档
			slog.Warn("slice不存在", "sliceID", sliceID)
			return nil, model.ErrSliceNotFound
		}

		slog.Error("获取slice失败", "sliceID", sliceID, "error", err)
		return nil, fmt.Errorf("获取slice失败: %w", err)
	}
	return slice, nil
}

func (s *Service) ListSlices() ([]*model.SliceProfile, error) {
	slices, err := s.Store.ListSlice()
	if err != nil { // 为空时list不会返回错误
		slog.Error("获取slice列表失败", "error", err)
		return nil, fmt.Errorf("获取slice列表失败: %w", err)
	}

	return slices, nil
}
