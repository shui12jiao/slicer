package service

import (
	"errors"
	"log/slog"

	"go.mongodb.org/mongo-driver/mongo"
)

func isNotFoundError(err error) bool {
	if errors.Is(err, mongo.ErrNoDocuments) { // MongoDB为空文档
		slog.Debug("MongoDB没有文档", "error", err)
		return true
	} else if errors.Is(err, mongo.ErrNilDocument) { // MongoDB没有文档
		slog.Debug("MongoDB返回空文档", "error", err)
		return true
	} else {
		slog.Debug("MongoDB返回错误", "error", err)
		return false
	}
}
