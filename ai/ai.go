package ai

import (
	"fmt"
	"os"
	"slicer/controller"
	"slicer/util"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/qianfan"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"golang.org/x/net/context"
)

const (
	// AI模型类型
	DeepSeek = "deepseek"
	Qwen     = "qwen"
	Ark      = "ark"
	QianFan  = "qianfan"
)

type AI interface {
	// Strategy实现
	controller.Strategy
}

type GeneralAI struct {
	*StrategyAgent
}

func NewGeneralAI(config *util.Config) (AI, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 初始化模型
	cm, err := NewModel(ctx, config.AIConfig)
	if err != nil {
		return nil, fmt.Errorf("模型初始化失败: %w", err)
	}

	// 初始化指标工具
	metrics, err := controller.NewMetrics(config.MonarchThanosURI)
	if err != nil {
		return nil, fmt.Errorf("指标工具初始化失败: %w", err)
	}
	metricsTool := &MetricsTool{
		Metrics: metrics,
	}

	sa, err := NewStrategyAgent(ctx, metricsTool, cm)
	if err != nil {
		return nil, err
	}
	return &GeneralAI{
		StrategyAgent: sa,
	}, nil
}

func NewModel(ctx context.Context, config util.AIConfig) (cm model.ToolCallingChatModel, err error) {
	var int2ptr = func(i int) *int {
		if i == 0 {
			return nil
		}
		return &i
	}
	var duration2ptr = func(d time.Duration) *time.Duration {
		if d == 0 {
			return nil
		}
		return &d
	}

	switch config.ModelType {
	case DeepSeek:
		cm, err = deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey: config.APIKey,
			Model:  config.Model,
			// 可选
			BaseURL:   config.BaseURL,
			Timeout:   config.Timeout,
			MaxTokens: config.MaxTokens,
		})
	case Qwen:
		cm, err = qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
			APIKey:  config.APIKey,
			Model:   config.Model,
			BaseURL: config.BaseURL,
			// 可选
			MaxTokens: int2ptr(config.MaxTokens),
			Timeout:   config.Timeout,
		})
	case Ark:
		cm, err = ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey: config.APIKey,
			Model:  config.Model,
			// 可选
			BaseURL:   config.BaseURL,
			MaxTokens: int2ptr(config.MaxTokens),
			Timeout:   duration2ptr(config.Timeout),
		})
	case QianFan:
		// QianFan模型用的环境变量
		os.Setenv("QIANFAN_ACCESS_KEY", config.APIKey)
		cm, err = qianfan.NewChatModel(ctx, &qianfan.ChatModelConfig{
			Model: config.Model,
			// 可选
			MaxCompletionTokens: int2ptr(config.MaxTokens),
			LLMRetryTimeout: func(d time.Duration) *float32 { // 使用单位为秒的浮点数, 库的代码注释真的依托
				if d == 0 {
					return nil
				}
				f := float32(d.Seconds())
				return &f
			}(config.Timeout),
		})
	default:
		return nil, fmt.Errorf("不支持的模型类型: %s", config.ModelType)
	}

	if err != nil {
		return nil, fmt.Errorf("模型初始化失败: %w", err)
	}
	return cm, nil
}
