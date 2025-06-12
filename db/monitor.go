package db

import (
	"context"
	"fmt"
	"slicer/model"
)

// Querier 接口实现
func (m *MongoDB) CreateMonitor(monitor *model.Monitor) error {
	_, err := m.insert(m.config.MonitorStoreName, monitor)

	if err != nil {
		return fmt.Errorf("插入Monitor失败：%w", err)
	}
	return nil
}

func (m *MongoDB) DeleteMonitor(id string) error {
	return m.delete(m.config.MonitorStoreName, id)
}

func (m *MongoDB) GetMonitor(id string) (*model.Monitor, error) {
	res := m.find(m.config.MonitorStoreName, id)

	monitor := new(model.Monitor)
	if err := res.Decode(monitor); err != nil {
		return nil, fmt.Errorf("查询Monitor失败：%w", err)
	}

	return monitor, nil
}

func (m *MongoDB) ListMonitor() ([]*model.Monitor, error) {
	// 获取所有 Monitor
	cursor, err := m.findAll(m.config.MonitorStoreName)
	if err != nil {
		return nil, fmt.Errorf("查询Monitor失败：%w", err)
	}
	defer cursor.Close(context.Background())

	var monitors []*model.Monitor
	if err := cursor.All(context.Background(), &monitors); err != nil {
		return nil, fmt.Errorf("查询Monitor失败：%w", err)
	}

	return monitors, nil
}
