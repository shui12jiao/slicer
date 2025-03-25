package db

import (
	"context"
	"fmt"
	"slicer/model"
	"slicer/util"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	config   util.Config
	client   *mongo.Client // 连接客户端[1](@ref)
	database string        // 数据库名称
	timeout  time.Duration // 操作超时时间
}

// New 创建MongoDB实例（单例模式推荐）
func NewMongoDB(config util.Config, opts ...*options.ClientOptions) (*MongoDB, error) {
	timeout := time.Duration(config.MongoTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 合并连接选项[1](@ref)
	baseOpts := options.Client().ApplyURI(config.MongoURI).
		SetServerSelectionTimeout(timeout).
		SetMaxPoolSize(10)
	opts = append(opts, baseOpts)

	client, err := mongo.Connect(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// 验证连接
	if err = client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return &MongoDB{
		config:   config,
		client:   client,
		database: config.MongoDBName,
		timeout:  timeout,
	}, nil
}

// Querier 接口实现
func (m *MongoDB) CreateSlice(slice model.SliceAndAddress) (model.SliceAndAddress, error) {
	res, err := m.insert(m.config.SliceStoreName, slice)

	// 获取插入的ID
	slice.ID = res.InsertedID.(primitive.ObjectID)
	if err != nil {
		return slice, fmt.Errorf("插入Slice失败：%w", err)
	}
	return slice, nil
}

func (m *MongoDB) DeleteSlice(id string) error {
	return m.delete(m.config.SliceStoreName, id)
}

func (m *MongoDB) GetSlice(id string) (model.SliceAndAddress, error) {
	res := m.find(m.config.SliceStoreName, id)

	var slice model.SliceAndAddress
	if err := res.Decode(&slice); err != nil {
		return slice, fmt.Errorf("查询Slice失败：%w", err)
	}

	return slice, nil
}

func (m *MongoDB) GetSliceBySliceID(sliceID string) (model.SliceAndAddress, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	var sst int
	var sd string
	_, err := fmt.Sscanf(sliceID, "%d-%s", &sst, &sd)
	if err != nil {
		return model.SliceAndAddress{}, fmt.Errorf("SliceID格式错误：%w", err)
	}

	// 查询 Slice
	res := m.client.Database(m.database).Collection(m.config.SliceStoreName).FindOne(ctx, primitive.M{"slice.sst": sst, "slice.sd": sd})
	var slice model.SliceAndAddress
	if err := res.Decode(&slice); err != nil {
		return slice, fmt.Errorf("查询Slice失败：%w", err)
	}

	return slice, nil
}

func (m *MongoDB) ListSlice() ([]model.SliceAndAddress, error) {
	// 获取所有 Slice
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	cursor, err := m.client.Database(m.database).Collection(m.config.SliceStoreName).Find(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("查询Slice失败：%w", err)
	}
	defer cursor.Close(ctx)

	var slices []model.SliceAndAddress
	if err = cursor.All(ctx, &slices); err != nil {
		return nil, fmt.Errorf("查询Slice失败：%w", err)
	}

	return slices, nil
}

func (m *MongoDB) ListSliceID() ([]string, error) {
	// 获取所有 Slice ID
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	cursor, err := m.client.Database(m.database).Collection(m.config.SliceStoreName).Find(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("查询Slice ID失败：%w", err)
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var slice model.SliceAndAddress
		if err = cursor.Decode(&slice); err != nil {
			return nil, fmt.Errorf("查询Slice ID失败：%w", err)
		}
		ids = append(ids, slice.SliceID())
	}

	return ids, nil
}

// 存储数据
func (m *MongoDB) insert(collection string, doc any) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	return m.client.Database(m.database).Collection(collection).InsertOne(ctx, doc)
}

func (m *MongoDB) delete(collection string, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("无效ID：%w", err)
	}

	_, err = m.client.Database(m.database).Collection(collection).DeleteOne(ctx, primitive.M{"_id": objID})
	return err
}

func (m *MongoDB) find(collection string, id string) *mongo.SingleResult {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil
	}

	return m.client.Database(m.database).Collection(collection).FindOne(ctx, primitive.M{"_id": objID})
}

// Close 关闭连接
func (m *MongoDB) Close() error {
	return m.client.Disconnect(context.Background())
}
