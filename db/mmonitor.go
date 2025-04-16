package db

import (
	"context"
	"fmt"
	"slicer/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Querier 接口实现
func (m *MongoDB) CreateMonitor(monitor model.Monitor) (model.Monitor, error) {
	res, err := m.insert(m.config.MonitorStoreName, monitor)

	// 获取插入的ID
	monitor.ID = res.InsertedID.(primitive.ObjectID)
	if err != nil {
		return monitor, fmt.Errorf("插入Monitor失败：%w", err)
	}
	return monitor, nil
}

func (m *MongoDB) DeleteMonitor(id string) error {
	return m.delete(m.config.MonitorStoreName, id)
}

func (m *MongoDB) GetMonitor(id string) (model.Monitor, error) {
	res := m.find(m.config.MonitorStoreName, id)

	var monitor model.Monitor
	if err := res.Decode(&monitor); err != nil {
		return monitor, fmt.Errorf("查询Monitor失败：%w", err)
	}

	return monitor, nil
}

func (m *MongoDB) ListMonitor() ([]model.Monitor, error) {
	// 获取所有 Monitor
	cursor, err := m.findAll(m.config.MonitorStoreName)
	if err != nil {
		return nil, fmt.Errorf("查询Monitor失败：%w", err)
	}
	defer cursor.Close(context.Background())

	var monitors []model.Monitor
	if err := cursor.All(context.Background(), &monitors); err != nil {
		return nil, fmt.Errorf("查询Monitor失败：%w", err)
	}

	return monitors, nil
}
