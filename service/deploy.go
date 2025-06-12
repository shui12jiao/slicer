package service

import (
	"log/slog"
	"slicer/model"
)

// // 更新部署
// err = s.kubeClient.Deploy(curDeploy, s.config.Namespace)
// if err != nil {
// 	slog.Error("更新deploy部署失败", "sliceID", sliceID, "error", err)
// 	http.Error(w, "更新deploy部署失败", http.StatusInternalServerError)
// 	return
// }

func (s *Service) UpdateDeploy(sliceID string, deploy model.Deploy) (model.Deploy, error) {
	// 检查是否存在Slice
	slice, err := s.GetSlice(sliceID)
	if err != nil {
		return model.Deploy{}, err
	}

	// 定义一个回滚栈，用于记录需要回滚的操作
	var rollbackFuncs []func()

	// 在函数退出时，根据是否出错决定是否执行回滚
	defer func() {
		if err != nil {
			slog.Debug("执行回滚操作")
			for i := len(rollbackFuncs) - 1; i >= 0; i-- {
				rollbackFuncs[i]()
			}
		}
	}()

	oldDeploy := slice.Deploy.Clone() // 保存旧的Deploy以便回滚
	// 更新Slice的Deploy
	if err = slice.Deploy.Update(deploy); err != nil {
		slog.Error("更新Slice Deploy失败", "sliceID", sliceID, "error", err)
		return model.Deploy{}, err
	}

	// 更新SliceProfile存储
	if err := s.Store.UpdateSlice(slice); err != nil {
		slog.Error("更新Slice Deploy失败", "sliceID", sliceID, "error", err)
		return model.Deploy{}, err
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		slice.Deploy = oldDeploy // 恢复旧的Deploy
		if rollbackErr := s.Store.UpdateSlice(slice); rollbackErr != nil {
			slog.Error("回滚Slice失败", "sliceID", sliceID, "error", rollbackErr)
		}
	})

	// 更新Kubernetes部署
	if err = s.KubeClient.Deploy(deploy,
		sliceID,
		s.Open5gs.HelmReleasePrefix+sliceID+"-upf",
		s.Open5gs.Namespace); err != nil {
		slog.Error("Deployment更新失败", "sliceID", sliceID, "error", err, "namespace", s.Open5gs.Namespace)
	}
	rollbackFuncs = append(rollbackFuncs, func() {
		if rollbackErr := s.KubeClient.Deploy(
			oldDeploy,
			sliceID,
			s.Open5gs.HelmReleasePrefix+sliceID+"-upf",
			s.Open5gs.Namespace); rollbackErr != nil {
			slog.Error("回滚Deployment失败", "sliceID", sliceID, "error", rollbackErr, "namespace", s.Open5gs.Namespace)
		}
	})

	// 返回更新后的Deploy对象
	slog.Info("更新Deploy成功", "sliceID", sliceID, "deploy", deploy)
	return deploy, nil
}
