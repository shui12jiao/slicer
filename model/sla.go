package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// SLA 包含带宽,延迟,可用性
type SLA struct {
	ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SliceID string             `json:"slice_id"`

	// 上行带宽
	UpBandwidth float64 `json:"up_bandwidth"` // 单位Mbps 例如 "100Mbps" 为 100
	// 下行带宽
	DownBandwidth float64 `json:"down_bandwidth"` // 单位Mbps 例如 "100Mbps"
	// 延迟
	Latency float64 `json:"latency"` // 单位ms 例如 "50ms" 为 50
	// 可用性
	Availability float64 `json:"availability"` // 例如 "99.9%" 为 99.9
}
