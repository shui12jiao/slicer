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
	MonitorTimeout              uint8

	// for mongodb
	MongoURI     string
	MongoDBName  string
	MongoTimeout uint8 // 单位秒

	// for kubernetes client
	Namespace      string
	KubeconfigPath string

	// for http server
	HTTPServerAddress string
	SliceStoreName    string
	KubeStoreName     string
	MonitorStoreName  string

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
		KubeStoreName:     MustGetEnvString("KUBE_STORE_NAME"), // 目前未使用
		MonitorStoreName:  MustGetEnvString("MONITOR_STORE_NAME"),

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
