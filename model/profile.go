package model

import (
	"fmt"
	"net"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SliceProfile struct {
	ID     primitive.ObjectID `json:"id" yaml:"id" bson:"_id,omitempty"`
	Slice  `json:"slice" yaml:"slice"`
	Deploy `json:"deploy" yaml:"deploy"`
	SLA    `json:"sla" yaml:"sla"`

	IsMonitored  bool `json:"is_monitored,omitempty" yaml:"is_monitored,omitempty"`
	AddressValue `json:"address_value,omitempty" yaml:"address_value,omitempty"`
}

// 格式均为x.x.x.x/x
type AddressValue struct {
	// Subnet是子网
	SessionSubnets []Subnet
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

type Subnet string // 10.41.0.0/16

func (s Subnet) Gateway() string { // 10.41.0.1
	_, ipnet, _ := net.ParseCIDR(string(s))
	ip := ipnet.IP.To4()
	ip[3] = 1 // 主机位设为1（如10.41.0.1）
	return ip.String()
}

func (s Subnet) GatewayWithCIDR() string { //10.41.0.1/16
	_, ipnet, _ := net.ParseCIDR(string(s))
	ip := ipnet.IP.To4()
	ip[3] = 1 // 主机位设为1（如10.41.0.1）
	maskSize, _ := ipnet.Mask.Size()
	return fmt.Sprintf("%s/%d", ip, maskSize)
}

func (s Subnet) String() string {
	return string(s)
}

func (s Subnet) Mask() int { // 返回长度， 如10.41.0.0/16返回16
	_, ipnet, _ := net.ParseCIDR(string(s))
	maskSize, _ := ipnet.Mask.Size()
	return maskSize
}
