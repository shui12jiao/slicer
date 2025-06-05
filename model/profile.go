package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SliceProfile struct {
	ID     primitive.ObjectID `json:"id" yaml:"id" bson:"_id,omitempty"`
	Slice  `json:"slice" yaml:"slice"`
	Deploy `json:"kube_config" yaml:"kube_config"`
	SLA    `json:"sla" yaml:"sla"`

	IsMonitored  bool `json:"is_monitored,omitempty" yaml:"is_monitored,omitempty"`
	AddressValue `json:"address_value,omitempty" yaml:"address_value,omitempty"`
}

// 格式均为x.x.x.x/x
type AddressValue struct {
	// Subnet是子网
	SessionSubnets []string
	// Addr是地址
	UPFN3Addr string
	UPFN4Addr string
	SMFN3Addr string
	SMFN4Addr string
}

func (av *AddressValue) IsEmpty() bool {
	return len(av.SessionSubnets) == 0 && av.UPFN3Addr == "" && av.UPFN4Addr == "" &&
		av.SMFN3Addr == "" && av.SMFN4Addr == ""
}
