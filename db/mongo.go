package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Document struct {
	ID   string `bson:"_id"`
	Data []byte `bson:"data"`
}

type MongoDB struct {
	client   *mongo.Client // 连接客户端[1](@ref)
	database string        // 数据库名称
	timeout  time.Duration // 操作超时时间
}

// New 创建MongoDB实例（单例模式推荐）
func NewMongoDB(uri, dbName string, timeout time.Duration, opts ...*options.ClientOptions) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 合并连接选项[1](@ref)
	baseOpts := options.Client().ApplyURI(uri).
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
		client:   client,
		database: dbName,
		timeout:  timeout,
	}, nil
}

func (m *MongoDB) Set(place, key string, value []byte) error {
	doc := &Document{
		ID:   key,
		Data: value,
	}
	return m.insert(place, doc)
}

func (m *MongoDB) Get(place, key string) ([]byte, error) {
	doc, error := m.find(place, key)
	if error != nil {
		return nil, error
	}
	return doc.Data, nil
}

func (m *MongoDB) Delete(place, key string) error {
	return m.delete(place, key)
}

func (m *MongoDB) Update(place, key string, value []byte) error {
	doc := &Document{
		ID:   key,
		Data: value,
	}
	return m.update(place, doc)
}

// 存储数据
func (m *MongoDB) insert(collection string, doc *Document) error {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	_, err := m.client.Database(m.database).Collection(collection).InsertOne(ctx, doc)
	return err
}

// 查询单/多条数据
// func (m *MongoDB) findAll(collection string, filter interface{}) ([]*Document, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
// 	defer cancel()

// 	cursor, err := m.client.Database(m.database).Collection(collection).Find(ctx, filter)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var docs []*Document
// 	if err = cursor.All(ctx, &docs); err != nil {
// 		return nil, err
// 	}

// 	return docs, nil
// }

func (m *MongoDB) find(collection string, id string) (*Document, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	doc := new(Document)
	err := m.client.Database(m.database).Collection(collection).FindOne(ctx, primitive.M{"_id": id}).Decode(doc)
	return doc, err
}

// func (m *MongoDB) deleteAll(collection string, filter interface{}) error {
// 	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
// 	defer cancel()

// 	_, err := m.client.Database(m.database).Collection(collection).DeleteOne(ctx, filter)
// 	return err
// }

func (m *MongoDB) delete(collection string, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	_, err := m.client.Database(m.database).Collection(collection).DeleteOne(ctx, primitive.M{"_id": id})
	return err
}

func (m *MongoDB) update(collection string, doc *Document) error {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	_, err := m.client.Database(m.database).Collection(collection).UpdateOne(ctx, primitive.M{"_id": doc.ID}, primitive.M{"$set": doc})
	return err
}

// Close 关闭连接
func (m *MongoDB) Close() error {
	return m.client.Disconnect(context.Background())
}
