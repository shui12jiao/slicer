package util

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// for monitor
	MonarchThanosURI            string
	MonarchRequestTranslatorURI string
	MonarchMonitoringInterval   uint8
	MonitorTimeout              uint8

	// for mongodb
	MongoURI     string
	MongoDBName  string
	MongoTimeout uint8 // 单位秒

	// for kubernetes client
	Namespace        string
	MonitorNamespace string
	KubeconfigPath   string

	// for http server
	HTTPServerAddress string
	SliceStoreName    string
	KubeStoreName     string
	MonitorStoreName  string
	PlayStoreName     string
	SLAStoreName      string

	// for render
	TemplatePath string

	// for ipam
	N3Network           string
	N4Network           string
	SessionNetwork      string
	SessionSubnetLength uint8
	IPAMTimeout         uint8 // 单位秒
}

func LoadConfig() Config {
	return Config{
		// for monitor
		MonarchThanosURI:            MustGetEnvString("MONARCH_THANOS_URL"),
		MonarchRequestTranslatorURI: MustGetEnvString("MONARCH_REQUEST_TRANSLATOR_URI"),
		MonarchMonitoringInterval:   MustGetEnvUInt8("MONARCH_MONITORING_INTERVAL"),
		MonitorTimeout:              MustGetEnvUInt8("MONITOR_TIMEOUT"),

		// for mongodb
		MongoURI:     MustGetEnvString("MONGO_URI"),
		MongoDBName:  MustGetEnvString("MONGO_DB_NAME"),
		MongoTimeout: MustGetEnvUInt8("MONGO_TIMEOUT"),

		// for kubernetes client
		Namespace:        MustGetEnvString("NAMESPACE"),         //用于open5gs的namespace
		MonitorNamespace: MustGetEnvString("MONITOR_NAMESPACE"), //监控系统所在的namespace

		KubeconfigPath: os.Getenv("KUBECONFIG_PATH"), // kubeconfig文件路径,可为空,如果不设置则使用集群内配置

		// for http server
		HTTPServerAddress: MustGetEnvString("HTTP_SERVER_ADDRESS"),
		SliceStoreName:    "slice",
		KubeStoreName:     "kube",
		MonitorStoreName:  "monitor",
		PlayStoreName:     "play",
		SLAStoreName:      "sla",

		// for render
		TemplatePath: MustGetEnvString("TEMPLATE_PATH"),

		// for ipam
		N3Network:           MustGetEnvString("N3_NETWORK"),
		N4Network:           MustGetEnvString("N4_NETWORK"),
		SessionNetwork:      MustGetEnvString("SESSION_NETWORK"),
		SessionSubnetLength: MustGetEnvUInt8("SESSION_SUBNET_LENGTH"),
		IPAMTimeout:         MustGetEnvUInt8("IPAM_TIMEOUT"),
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
