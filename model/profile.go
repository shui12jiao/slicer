package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type SliceProfile struct {
	ID     primitive.ObjectID `json:"id" yaml:"id" bson:"_id,omitempty"`
	Slice  `json:"slice" yaml:"slice"`
	Deploy `json:"kube_config" yaml:"kube_config"`
	SLA    `json:"sla" yaml:"sla"`

	IsMonitored bool `json:"is_monitored,omitempty" yaml:"is_monitored,omitempty"`
}
