package service

import (
	"log/slog"
	"slicer/model"
)

func (s *Service) UpdateSLA(sliceID string, sla model.SLA) (model.SLA, error) {
	// 检查是否存在Slice
	slice, err := s.GetSlice(sliceID)
	if err != nil {
		return model.SLA{}, err
	}

	oldSLA := slice.SLA // 保存旧的SLA以便回滚, floadt类型无需深拷贝
	slice.SLA = sla     // 更新Slice的SLA
	// 更新SliceProfile
	if _, err := s.Store.UpdateSlice(slice); err != nil {
		slog.Error("更新Slice SLA失败", "sliceID", sliceID, "error", err)
		// 回滚slice
		slice.SLA = oldSLA // 恢复旧的SLA
		if _, rollbackErr := s.Store.UpdateSlice(slice); rollbackErr != nil {
			slog.Error("回滚Slice失败", "sliceID", sliceID, "error", rollbackErr)
			return model.SLA{}, rollbackErr
		}
		return model.SLA{}, err
	}

	return sla, nil
}
