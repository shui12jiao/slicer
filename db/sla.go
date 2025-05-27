package db

import (
	"context"
	"fmt"
	"slicer/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Querier 接口实现
func (m *MongoDB) CreateSLA(sla model.SLA) (model.SLA, error) {
	res, err := m.insert(m.config.SLAStoreName, sla)

	// 获取插入的ID
	sla.ID = res.InsertedID.(primitive.ObjectID)
	if err != nil {
		return sla, fmt.Errorf("插入SLA失败：%w", err)
	}
	return sla, nil
}

func (m *MongoDB) DeleteSLA(id string) error {
	return m.delete(m.config.SLAStoreName, id)
}

func (m *MongoDB) GetSLA(id string) (model.SLA, error) {
	res := m.find(m.config.SLAStoreName, id)

	var sla model.SLA
	if err := res.Decode(&sla); err != nil {
		return sla, fmt.Errorf("查询SLA失败：%w", err)
	}

	return sla, nil
}

func (m *MongoDB) GetSLABySliceID(sliceID string) (model.SLA, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	var sst int
	var sd string
	_, err := fmt.Sscanf(sliceID, "%d-%s", &sst, &sd)
	if err != nil {
		return model.SLA{}, fmt.Errorf("SLAID格式错误：%w", err)
	}

	// 查询 SLA
	res := m.client.Database(m.database).Collection(m.config.SLAStoreName).FindOne(ctx, primitive.M{"slice.sst": sst, "slice.sd": sd})
	var sla model.SLA
	if err := res.Decode(&sla); err != nil {
		return sla, fmt.Errorf("查询SLA失败：%w", err)
	}

	return sla, nil
}

func (m *MongoDB) ListSLA() ([]model.SLA, error) {
	// 获取所有 SLA
	cursor, err := m.findAll(m.config.SLAStoreName)
	if err != nil {
		return nil, fmt.Errorf("查询SLA失败：%w", err)
	}

	defer cursor.Close(context.Background())
	var slas []model.SLA
	for cursor.Next(context.Background()) {
		var sla model.SLA
		if err = cursor.Decode(&sla); err != nil {
			return nil, fmt.Errorf("查询SLA失败：%w", err)
		}
		slas = append(slas, sla)
	}
	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("查询SLA失败：%w", err)
	}
	return slas, nil
}

func (m *MongoDB) UpdateSLA(sla model.SLA) (model.SLA, error) {
	// 更新SLA
	res, err := m.update(m.config.SLAStoreName, sla.ID, sla)
	if err != nil {
		return sla, fmt.Errorf("更新SLA失败：%w", err)
	}

	// 获取更新的ID
	sla.ID = res.UpsertedID.(primitive.ObjectID)
	return sla, nil
}
