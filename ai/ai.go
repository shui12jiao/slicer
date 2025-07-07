package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"slicer/controller"
	"slicer/util"
	"time"

	qianfanmodel "github.com/baidubce/bce-qianfan-sdk/go/qianfan"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/qianfan"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"golang.org/x/net/context"
)

const (
	// AI模型类型
	Ollama   = "ollama"
	DeepSeek = "deepseek"
	Qwen     = "qwen"
	Ark      = "ark"
	QianFan  = "qianfan"
)

type AI interface {
	// Strategy实现
	controller.Strategy
	// Debug方法
	Ping(ctx context.Context) (time.Duration, error)
}

type GeneralAI struct {
	Model model.ToolCallingChatModel
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
		Model:         cm,
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
	case Ollama:
		cm, err = ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			BaseURL: config.BaseURL,   // Ollama 服务地址
			Timeout: config.AITimeout, // 请求超时时间
			Model:   config.Model,
			Format:  json.RawMessage(`"json"`), // 输出格式（可选）
		})
	case DeepSeek:
		cm, err = deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey: config.APIKey,
			Model:  config.Model,
			// 可选
			BaseURL:   config.BaseURL,
			Timeout:   config.AITimeout,
			MaxTokens: config.MaxTokens,
			// 设置为json格式
			ResponseFormatType: deepseek.ResponseFormatTypeJSONObject, // 响应格式
		})
	case Qwen:
		cm, err = qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
			APIKey:  config.APIKey,
			Model:   config.Model,
			BaseURL: config.BaseURL,
			// 可选
			MaxTokens: int2ptr(config.MaxTokens),
			Timeout:   config.AITimeout,
			// 设置为json格式，不设置json schema 未来可能会使用
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
		})
	case Ark:
		cm, err = ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey: config.APIKey,
			Model:  config.Model,
			// 可选
			BaseURL:   config.BaseURL,
			MaxTokens: int2ptr(config.MaxTokens),
			Timeout:   duration2ptr(config.AITimeout),
			// 设置为json格式
			ResponseFormat: &ark.ResponseFormat{
				Type: arkmodel.ResponseFormatJsonObject,
			},
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
			}(config.AITimeout),
			// 设置为json格式, 傻口百度又没文档，不知道对不对反正用不着，其python文档sdk写的txt，json，jsonl。
			ResponseFormat: &qianfanmodel.ResponseFormat{
				FormatType: "json",
			},
		})
	default:
		return nil, fmt.Errorf("不支持的模型类型: %s", config.ModelType)
	}

	if err != nil {
		return nil, fmt.Errorf("模型初始化失败: %w", err)
	}
	return cm, nil
}

func (g *GeneralAI) Ping(ctx context.Context) (time.Duration, error) {
	// 检查模型是否已初始化
	if g.Model == nil {
		return 0, fmt.Errorf("GeneralAI: 模型未初始化")
	}

	// 生成一个简单的消息来测试模型
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "这是一个测试消息，用于检查AI模型是否正常工作。",
		},
	}

	start := time.Now()

	// 调用模型生成响应
	_, err := g.Model.Generate(ctx, messages)
	if err != nil {
		return 0, fmt.Errorf("连接模型失败: %w", err)
	}

	return time.Since(start), nil
}
