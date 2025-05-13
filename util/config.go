package util

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type MongoConfig struct {
	MongoURI     string
	MongoDBName  string
	MongoTimeout time.Duration
}

type MonitorConfig struct {
	MonarchThanosURI            string
	MonarchRequestTranslatorURI string
	MonarchMonitoringInterval   uint8
	MonitorTimeout              time.Duration
}

type KubeConfig struct {
	Namespace        string
	MonitorNamespace string
	KubeconfigPath   string
}

type ServerConfig struct {
	HTTPServerAddress string
	SliceStoreName    string
	KubeStoreName     string
	MonitorStoreName  string
	PlayStoreName     string
	SLAStoreName      string
}

type IPAMConfig struct {
	N3Network           string
	N4Network           string
	SessionNetwork      string
	SessionSubnetLength uint8
	IPAMTimeout         time.Duration
}

type AIConfig struct {
	ModelType string
	Model     string
	APIKey    string
	// 可选
	BaseURL   string
	Timeout   time.Duration
	MaxTokens int
}

type Config struct {
	// for monitor
	MonitorConfig

	// for mongodb
	MongoConfig

	// for kubernetes client
	KubeConfig

	// for http server
	ServerConfig

	// for render
	TemplatePath string

	// for ipam
	IPAMConfig

	// for ai
	AIConfig
}

func LoadConfig() Config {
	return Config{
		// for monitor
		MonitorConfig: MonitorConfig{
			MonarchThanosURI:            MustGetEnv("MONARCH_THANOS_URL"),
			MonarchRequestTranslatorURI: MustGetEnv("MONARCH_REQUEST_TRANSLATOR_URI"),
			MonarchMonitoringInterval:   String2Uint8(MustGetEnv("MONARCH_MONITORING_INTERVAL")),
			MonitorTimeout:              String2Duration(MustGetEnv("MONITOR_TIMEOUT")),
		},

		// for mongodb
		MongoConfig: MongoConfig{
			MongoURI:     MustGetEnv("MONGO_URI"),
			MongoDBName:  MustGetEnv("MONGO_DB_NAME"),
			MongoTimeout: String2Duration(MustGetEnv("MONGO_TIMEOUT")),
		},

		// for kubernetes client
		KubeConfig: KubeConfig{
			Namespace:        MustGetEnv("NAMESPACE"),         //用于open5gs的namespace
			MonitorNamespace: MustGetEnv("MONITOR_NAMESPACE"), //监控系统所在的namespace
			KubeconfigPath:   os.Getenv("KUBECONFIG_PATH"),    // kubeconfig文件路径,可为空,如果不设置则使用集群内配置
		},

		// for http server
		ServerConfig: ServerConfig{
			HTTPServerAddress: MustGetEnv("HTTP_SERVER_ADDRESS"),
			SliceStoreName:    "slice",
			KubeStoreName:     "kube",
			MonitorStoreName:  "monitor",
			PlayStoreName:     "play",
			SLAStoreName:      "sla",
		},

		// for render
		TemplatePath: MustGetEnv("TEMPLATE_PATH"),

		// for ipam
		IPAMConfig: IPAMConfig{
			N3Network:           MustGetEnv("N3_NETWORK"),
			N4Network:           MustGetEnv("N4_NETWORK"),
			SessionNetwork:      MustGetEnv("SESSION_NETWORK"),
			SessionSubnetLength: String2Uint8(MustGetEnv("SESSION_SUBNET_LENGTH")),
			IPAMTimeout:         String2Duration(MustGetEnv("IPAM_TIMEOUT")),
		},

		// for ai
		AIConfig: AIConfig{
			ModelType: MustGetEnv("MODEL_TYPE"),
			Model:     MustGetEnv("MODEL"),
			APIKey:    MustGetEnv("API_KEY"),
			// 可选
			BaseURL:   GetEnv("BASE_URL"),
			Timeout:   String2Duration(GetEnv("AI_TIMEOUT")),
			MaxTokens: String2Int(GetEnv("AI_MAX_TOKENS")),
		},
	}
}

func GetEnv(key string) string {
	s := os.Getenv(key)
	if s == "" {
		slog.Info(fmt.Sprintf("变量 %s 为空", key))
	}
	return s
}

func MustGetEnv(key string) string {
	s := os.Getenv(key)
	if s == "" {
		slog.Error(fmt.Sprintf("变量 %s 为空", key))
		os.Exit(1)
	}
	return s
}

func String2Uint8(s string) uint8 {
	i, err := strconv.Atoi(s)
	if err != nil {
		slog.Warn(fmt.Sprintf("变量 %s 转换失败", s))
		os.Exit(1)
	}
	if i < 0 || i > 255 {
		slog.Warn(fmt.Sprintf("变量 %s 超出范围", s))
		// os.Exit(1)
	}
	return uint8(i)
}

func String2Int(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		slog.Warn(fmt.Sprintf("变量 %s 转换失败", s))
		// os.Exit(1)
	}
	return i
}

func String2Duration(s string) time.Duration {
	// 检查是否为纯数字
	if seconds, err := strconv.Atoi(s); err == nil {
		// 将纯数字视为秒
		return time.Duration(seconds) * time.Second
	} else {
		d, err := time.ParseDuration(s)
		if err != nil {
			slog.Warn(fmt.Sprintf("变量 %s 转换失败", s))
			// os.Exit(1)
		}
		return d
	}
}
