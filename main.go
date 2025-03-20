package main

import (
	"fmt"
	"os"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/monitor"
	"slicer/render"
	"slicer/server"
	"slicer/util"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	config := util.Config{
		// for monitor
		MonarchThanosURI:            MustGetEnvString("MONARCH_THANOS_URL"),
		MonarchRequestTranslatorURI: MustGetEnvString("MONARCH_REQUEST_TRANSLATOR_URI"),
		MonitorTimeout:              MustGetEnvUInt8("MONITOR_TIMEOUT"),

		// for mongodb
		MongoURI:     MustGetEnvString("MONGO_URI"),
		MongoDBName:  MustGetEnvString("MONGO_DB_NAME"),
		MongoTimeout: MustGetEnvUInt8("MONGO_TIMEOUT"),

		// for kubernetes client
		Namespace:      MustGetEnvString("NAMESPACE"),
		KubeconfigPath: MustGetEnvString("KUBECONFIG_PATH"),

		// for http server
		HTTPServerAddress: MustGetEnvString("HTTP_SERVER_ADDRESS"),
		SliceStoreName:    MustGetEnvString("SLICE_STORE_NAME"),
		KubeStoreName:     MustGetEnvString("KUBE_STORE_NAME"),

		// for render
		TemplatePath: MustGetEnvString("TEMPLATE_PATH"),

		// for ipam
		N3Network:           MustGetEnvString("N3_NETWORK"),
		N4Network:           MustGetEnvString("N4_NETWORK"),
		SessionNetwork:      MustGetEnvString("SESSION_NETWORK"),
		SessionSubnetLength: MustGetEnvUInt8("SESSION_SUBNET_LENGTH"),
		IPAMTimeout:         MustGetEnvUInt8("IPAM_TIMEOUT"),
	}

	// 初始化monitor监控系统交互组件
	monitor, err := monitor.NewMonitor(config)

	// 连接数据库
	store, err := db.NewMongoDB(config.MongoURI, config.MongoDBName, time.Duration(config.MongoTimeout)*time.Second)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to mongodb: %v", err))
	}

	// 初始化渲染器
	render := render.NewRender(config)

	// 初始化Kubernetes客户端
	kubeclient, err := kubeclient.NewKubeClient(config.KubeconfigPath)
	if err != nil {
		panic(fmt.Sprintf("failed to create kubeclient: %v", err))
	}

	// 初始化IPAM
	ipam, err := db.NewIPAM(config)
	if err != nil {
		panic(fmt.Sprintf("failed to create ipam: %v", err))
	}

	// 初始化Server
	server := server.NewServer(config, monitor, store, ipam, render, kubeclient)

	server.Start()
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
