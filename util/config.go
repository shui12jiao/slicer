package util

import (
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type MongoConfig struct {
	MongoURI         string        `envconfig:"MONGO_URI" required:"true"`
	MongoDBName      string        `envconfig:"MONGO_DB_NAME" required:"true"`
	MongoTimeout     time.Duration `envconfig:"MONGO_TIMEOUT" default:"15s"`
	SliceStoreName   string        `envconfig:"SLICE_STORE_NAME" default:"slice"`
	MonitorStoreName string        `envconfig:"MONITOR_STORE_NAME" default:"monitor"`
}

type MonitorConfig struct {
	MonarchThanosURI            string        `envconfig:"MONARCH_THANOS_URL" required:"true"`
	MonarchRequestTranslatorURI string        `envconfig:"MONARCH_REQUEST_TRANSLATOR_URI" required:"true"`
	MonarchMonitoringInterval   uint8         `envconfig:"MONARCH_MONITORING_INTERVAL" required:"true"`
	MonitorTimeout              time.Duration `envconfig:"MONITOR_TIMEOUT" default:"30s"`
}

type KubeConfig struct {
	KubeconfigPath   string        `envconfig:"KUBECONFIG_PATH"` // 集群内使用时为空
	Namespace        string        `envconfig:"NAMESPACE" required:"true"`
	MonitorNamespace string        `envconfig:"MONITOR_NAMESPACE" required:"true"`
	HelmDriver       string        `envconfig:"HELM_DRIVER" default:"configmap"`
	HelmTimeout      time.Duration `envconfig:"HELM_TIMEOUT" default:"5m"`
}

type ServerConfig struct {
	HTTPServerAddress string `envconfig:"HTTP_SERVER_ADDRESS" required:"true"`
}

type IPAMConfig struct {
	N3Network           string        `envconfig:"N3_NETWORK" required:"true" default:"10.10.3.0/24"`
	N4Network           string        `envconfig:"N4_NETWORK" required:"true" default:"10.10.4.0/24"`
	SessionNetwork      string        `envconfig:"SESSION_NETWORK" required:"true" default:"10.32.0.0/12"`
	SessionSubnetLength uint8         `envconfig:"SESSION_SUBNET_LENGTH" default:"16"`
	IPAMTimeout         time.Duration `envconfig:"IPAM_TIMEOUT" default:"1m"`
}

type AIConfig struct {
	ModelType string        `envconfig:"MODEL_TYPE" required:"true"`
	Model     string        `envconfig:"MODEL" required:"true"`
	APIKey    string        `envconfig:"API_KEY" required:"true"`
	BaseURL   string        `envconfig:"BASE_URL"`
	AITimeout time.Duration `envconfig:"AI_TIMEOUT" default:"30s"`
	MaxTokens int           `envconfig:"AI_MAX_TOKENS"`
}

type ServiceConfig struct {
	CommonChartPath   string `envconfig:"COMMON_CHART_PATH" required:"true"`
	SliceChartPath    string `envconfig:"SLICE_CHART_PATH" required:"true"`
	HelmReleasePrefix string `envconfig:"HELM_RELEASE_PREFIX" default:"open5gs"`
}

type Config struct {
	MonitorConfig
	MongoConfig
	KubeConfig
	ServerConfig
	IPAMConfig
	AIConfig
	ServiceConfig
}

func LoadConfig(path string) *Config {
	// 加载环境变量配置
	err := godotenv.Load(path)
	if err != nil {
		slog.Error("加载环境变量配置失败", "error", err)
		os.Exit(1)
	}

	var cfg Config
	err = envconfig.Process("", &cfg)
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	return &cfg
}
