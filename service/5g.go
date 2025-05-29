package service

import (
	"slicer/kube/value"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Open5gs 定义Open5GS的Helm Chart相关信息
// 用于在Kubernetes集群中部署和管理Open5GS
type Open5gs struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Namespace       string             // Open5GS的命名空间
	HelmChart       string             // Open5GS的Helm Chart路径
	HelmValues      value.Open5gs      // Open5GS的Helm Values
	HelmReleaseName string             // Open5GS的Helm Release名称
	SliceIDs        []string           // 切片ID列表
}

func NewOpen5gs(namespace, helmChart, helmReleaseName string, values value.Open5gs) *Open5gs {
	return &Open5gs{
		Namespace:       namespace,
		HelmChart:       helmChart,
		HelmValues:      values,
		HelmReleaseName: helmReleaseName,
	}
}
