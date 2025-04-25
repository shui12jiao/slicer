package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/yaml.v3"
)

type SliceAndAddress struct {
	ID primitive.ObjectID `json:"id" yaml:"id" bson:"_id,omitempty"`
	Slice
	AddressValue
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

func (s *SliceAndAddress) ToYAML() ([]byte, error) {
	return yaml.Marshal(s)
}

func (s *SliceAndAddress) FromYAML(data []byte) error {
	return yaml.Unmarshal(data, s)
}
