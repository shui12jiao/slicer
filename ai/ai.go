package ai

import (
	"fmt"
	"slicer/controller"
	"slicer/util"
	"time"

	"errors"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/model"
	"golang.org/x/net/context"
)

type AI interface {
	// Strategy实现
	controller.Strategy
}

type GeneralAI struct {
	*StrategyAgent
}

func NewGeneralAI(config util.Config) (AI, error) {
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

func NewModel(ctx context.Context, config util.AIConfig) (model.ToolCallingChatModel, error) {
	switch config.Model {
	case "deepseek":
		cm, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey: config.APIKey,
			Model:  config.Model,
		})
		if err != nil {
			return nil, err
		}
		return cm, nil
	default:
		return nil, errors.New("不支持的模型类型")
	}
}
