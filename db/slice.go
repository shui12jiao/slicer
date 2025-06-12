package db

import (
	"context"
	"fmt"
	"slicer/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Querier 接口实现
func (m *MongoDB) UpdateSlice(slice *model.SliceProfile) error {
	_, err := m.update(m.config.SliceStoreName, slice.ID, slice, true)
	if err != nil {
		return fmt.Errorf("更新Slice失败：%w", err)
	}

	return nil
}

func (m *MongoDB) CreateSlice(slice *model.SliceProfile) error {
	_, err := m.insert(m.config.SliceStoreName, slice)

	// 如果ID为空，mongo会自动赋值，无需手动设置
	if err != nil {
		return fmt.Errorf("插入Slice失败：%w", err)
	}
	return nil
}

func (m *MongoDB) DeleteSlice(id string) error {
	return m.delete(m.config.SliceStoreName, id)
}

func (m *MongoDB) GetSlice(id string) (*model.SliceProfile, error) {
	res := m.find(m.config.SliceStoreName, id)

	slice := new(model.SliceProfile)
	if err := res.Decode(slice); err != nil {
		return nil, fmt.Errorf("查询Slice失败：%w", err)
	}

	return slice, nil
}

func (m *MongoDB) GetSliceBySliceID(sliceID string) (*model.SliceProfile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	var sst int
	var sd string
	_, err := fmt.Sscanf(sliceID, "%d-%s", &sst, &sd)
	if err != nil {
		return nil, fmt.Errorf("SliceID格式错误：%w", err)
	}

	// 查询 Slice
	res := m.client.Database(m.database).Collection(m.config.SliceStoreName).FindOne(ctx, primitive.M{"slice.sst": sst, "slice.sd": sd})
	slice := new(model.SliceProfile)
	if err := res.Decode(slice); err != nil {
		return nil, fmt.Errorf("查询Slice失败：%w", err)
	}

	return slice, nil
}

func (m *MongoDB) ListSlice() ([]*model.SliceProfile, error) {
	// 获取所有 Slice
	cursor, err := m.findAll(m.config.SliceStoreName)
	if err != nil {
		return nil, fmt.Errorf("查询Slice失败：%w", err)
	}

	defer cursor.Close(context.Background())
	var slices []*model.SliceProfile
	if err := cursor.All(context.Background(), &slices); err != nil {
		return nil, fmt.Errorf("查询Slice失败：%w", err)
	}

	return slices, nil
}

func (m *MongoDB) ListSliceID() ([]string, error) {
	// 获取所有 Slice ID
	cursor, err := m.findAll(m.config.SliceStoreName)
	if err != nil {
		return nil, fmt.Errorf("查询Slice ID失败：%w", err)
	}
	defer cursor.Close(context.Background())

	var ids []string
	for cursor.Next(context.Background()) {
		var slice model.SliceProfile
		if err = cursor.Decode(&slice); err != nil {
			return nil, fmt.Errorf("查询Slice ID失败：%w", err)
		}
		ids = append(ids, slice.SliceID())
	}

	return ids, nil
}
