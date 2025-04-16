package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// SLA 包含带宽,延迟,可用性
type SLA struct {
	ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SliceID string             `json:"slice_id"`

	// 带宽
	Bandwidth string `json:"bandwidth"` // 例如 "100Mbps"
	// 延迟
	Latency string `json:"latency"` // 例如 "10ms"
	// 可用性
	Availability string `json:"availability"` // 例如 "99.9%"
}
