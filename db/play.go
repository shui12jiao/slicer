package db

import (
	"context"
	"fmt"
	"slicer/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Querier 接口实现
func (m *MongoDB) CreatePlay(play model.Play) (model.Play, error) {
	res, err := m.insert(m.config.PlayStoreName, play)

	// 获取插入的ID
	play.ID = res.InsertedID.(primitive.ObjectID)
	if err != nil {
		return play, fmt.Errorf("插入Play失败：%w", err)
	}
	return play, nil
}

func (m *MongoDB) DeletePlay(id string) error {
	return m.delete(m.config.PlayStoreName, id)
}

func (m *MongoDB) GetPlay(id string) (model.Play, error) {
	res := m.find(m.config.PlayStoreName, id)

	var play model.Play
	if err := res.Decode(&play); err != nil {
		return play, fmt.Errorf("查询Play失败：%w", err)
	}

	return play, nil
}

func (m *MongoDB) GetPlayBySliceID(sliceID string) (model.Play, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	var sst int
	var sd string
	_, err := fmt.Sscanf(sliceID, "%d-%s", &sst, &sd)
	if err != nil {
		return model.Play{}, fmt.Errorf("PlayID格式错误：%w", err)
	}

	// 查询 Play
	res := m.client.Database(m.database).Collection(m.config.PlayStoreName).FindOne(ctx, primitive.M{"slice.sst": sst, "slice.sd": sd})
	var play model.Play
	if err := res.Decode(&play); err != nil {
		return play, fmt.Errorf("查询Play失败：%w", err)
	}

	return play, nil
}

func (m *MongoDB) ListPlay() ([]model.Play, error) {
	// 获取所有 Play
	cursor, err := m.findAll(m.config.PlayStoreName)
	if err != nil {
		return nil, fmt.Errorf("查询Play失败：%w", err)
	}

	defer cursor.Close(context.Background())
	var plays []model.Play
	for cursor.Next(context.Background()) {
		var play model.Play
		if err = cursor.Decode(&play); err != nil {
			return nil, fmt.Errorf("查询Play失败：%w", err)
		}
		plays = append(plays, play)
	}
	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("查询Play失败：%w", err)
	}
	return plays, nil
}

func (m *MongoDB) UpdatePlay(play model.Play) (model.Play, error) {
	// 更新Play
	res, err := m.update(m.config.PlayStoreName, play.ID, play)
	if err != nil {
		return play, fmt.Errorf("更新Play失败：%w", err)
	}

	// 获取更新的ID
	play.ID = res.UpsertedID.(primitive.ObjectID)
	return play, nil
}
