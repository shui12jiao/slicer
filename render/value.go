package render

import (
	"fmt"
	"net"
)

type KpiCalc struct {
	SliceID   string // 切片ID
	ThanosURL string // Thanos地址
}

type MdeValue struct {
	SliceID  string // 切片ID
	Interval uint8  // 采集间隔
}

type SliceValue struct {
	ID  string // 切片ID= SST-SD
	SST string
	SD  string
}

type SessionValue struct {
	Subnet string // 10.41.0.0/16
	DNN    string
	Dev    string
}

func (s *SessionValue) Gateway() string { // 10.41.0.1/16
	_, ipnet, _ := net.ParseCIDR(s.Subnet)
	ip := ipnet.IP.To4()
	ip[3] = 1 // 主机位设为1（如10.41.0.1）
	maskSize, _ := ipnet.Mask.Size()
	return fmt.Sprintf("%s/%d", ip, maskSize)
}

type SessionValues = []SessionValue

type SmfConfigmapValue struct {
	SliceValue
	UPFN4Addr string
	SessionValues
}

type SmfDeploymentValue struct {
	SliceValue
	N4Addr string
	N3Addr string
}

type SmfServiceValue struct {
	SliceValue
}

type UpfConfigmapValue struct {
	SliceValue
	SessionValues
}

type UpfDeploymentValue struct {
	SliceValue
	N4Addr string
	N3Addr string
}
