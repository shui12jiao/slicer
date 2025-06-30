package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slicer/util"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	config   *util.Config
	client   *mongo.Client // 连接客户端[1](@ref)
	database string        // 数据库名称
	timeout  time.Duration // 操作超时时间
}

// New 创建MongoDB实例（单例模式推荐）
func NewMongoDB(config *util.Config, opts ...*options.ClientOptions) (*MongoDB, error) {
	timeout := config.MongoTimeout
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

// 更新
// 设置upsert为true时，如果文档不存在则插入新文档
func (m *MongoDB) update(collection string, objID primitive.ObjectID, doc any, upsert bool) (*mongo.UpdateResult, error) {
	// 检查 objID 是否为零值
	if objID.IsZero() {
		return nil, errors.New("无效的 ObjectID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	res, err := m.client.Database(m.database).Collection(collection).UpdateOne(ctx,
		primitive.M{"_id": objID}, bson.M{"$set": doc}, options.Update().SetUpsert(upsert))
	return res, err
}

// 存储数据
func (m *MongoDB) insert(collection string, doc any) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	res, err := m.client.Database(m.database).Collection(collection).InsertOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("插入数据失败：%w", err)
	}

	// 尝试将 MongoDB 生成的 ID 分配给文档中的 ObjectID 字段
	err = assignObjectID(doc, res.InsertedID)
	if err != nil {
		return nil, fmt.Errorf("分配 ObjectID 失败：%w", err)
	}

	return res, nil
}

func assignObjectID(doc any, insertedID any) error {
	v := reflect.ValueOf(doc)
	// 必须是可修改的指针类型
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return nil // 非结构体指针直接跳过
	}

	// 尝试转换 MongoDB 生成的 ID
	oid, ok := insertedID.(primitive.ObjectID)
	if !ok {
		return errors.New("InsertedID 非 ObjectID 类型")
	}

	// 遍历结构体字段
	s := v.Elem()
	for i := 0; i < s.NumField(); i++ {
		fieldValue := s.Field(i)

		// 跳过不可修改字段
		if !fieldValue.CanSet() {
			continue
		}

		// 检查字段类型是否为 primitive.ObjectID
		if fieldValue.Type() == reflect.TypeOf(primitive.ObjectID{}) {
			// 仅当字段为零值时赋值（避免覆盖已有值）
			if fieldValue.IsZero() {
				fieldValue.Set(reflect.ValueOf(oid))
				return nil // 找到第一个匹配字段即退出
			}
		}
	}
	return nil
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

func (m *MongoDB) findAll(collection string) (*mongo.Cursor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	// 注意必须使用 bson.D{}(primitive.M{}等空结构体均可以)，而非nil(表示不查询) 网上用例教程等已过时
	return m.client.Database(m.database).Collection(collection).Find(ctx, bson.D{})
}

// Close 关闭连接
func (m *MongoDB) Close() error {
	return m.client.Disconnect(context.Background())
}
