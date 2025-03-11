package main

import (
	"fmt"
	"os"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/render"
	"slicer/util"
	"strconv"
	"time"
)

func main() {
	config := util.Config{
		MongoURI:     MustGetEnvString("MONGO_URI"),
		MongoDBName:  MustGetEnvString("MONGO_DB_NAME"),
		MongoTimeout: MustGetEnvUInt8("MONGO_TIMEOUT"),

		Namespace:      MustGetEnvString("NAMESPACE"),
		KubeconfigPath: MustGetEnvString("KUBECONFIG_PATH"),

		HTTPServerAddress: MustGetEnvString("HTTP_SERVER_ADDRESS"),
		SliceStoreName:    MustGetEnvString("SLICE_STORE_NAME"),
		KubeStoreName:     MustGetEnvString("KUBE_STORE_NAME"),

		TemplatePath: MustGetEnvString("TEMPLATE_PATH"),

		N3Network:           MustGetEnvString("N3_NETWORK"),
		N4Network:           MustGetEnvString("N4_NETWORK"),
		SessionNetwork:      MustGetEnvString("SESSION_NETWORK"),
		SessionSubnetLength: MustGetEnvUInt8("SESSION_SUBNET_LENGTH"),
		IPAMTimeout:         MustGetEnvUInt8("IPAM_TIMEOUT"),
	}

	// 连接数据库
	store, err := db.NewMongoDB(config.MongoURI, config.MongoDBName, time.Duration(config.MongoTimeout)*time.Second)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to mongodb: %v", err))
	}

	// 初始化渲染器
	render, err := render.NewRender(config.TemplatePath)
	if err != nil {
		panic(fmt.Sprintf("failed to create render: %v", err))
	}

	// 初始化Kubernetes客户端
	kubeclient, err := kubeclient.NewKubeClient("")
	if err != nil {
		panic(fmt.Sprintf("failed to create kubeclient: %v", err))
	}

	// 初始化IPAM
	ipam, err := db.NewIPAM(config)
	if err != nil {
		panic(fmt.Sprintf("failed to create ipam: %v", err))
	}

}

func MustGetEnvUInt8(key string) uint8 {
	s := os.Getenv(key)
	val, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("invalid int for env %s: %v", key, err))
	}
	return uint8(val)
}

func MustGetEnvString(key string) string {
	s := os.Getenv(key)
	if s == "" {
		panic(fmt.Sprintf("env %s is empty", key))
	}
	return s
}
