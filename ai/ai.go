package ai

import (
	"context"
	"errors"
	"slicer/controller"
	sm "slicer/model"
	"slicer/util"
	"time"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type AI interface {
	// Strategy实现
	controller.Strategy
}

type GeneralAI struct {
	model  model.ToolCallingChatModel
	prompt string
	tools  []schema.ToolInfo
}

func NewGeneralAI(config util.AIConfig, prompt string, tools []schema.ToolInfo) (AI, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch config.ModelType {
	case "deepseek":
		cm, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey: config.APIKey,
			Model:  config.Model,
		})
		if err != nil {
			return nil, err
		}
		return &GeneralAI{
			model:  cm,
			prompt: prompt,
			tools:  tools,
		}, nil
	default:
		return nil, errors.New("不支持的模型类型")
	}
}

func (g *GeneralAI) Reconcile(current sm.Play, sla sm.SLA) (sm.Play, error) {
	// TODO
	return current, nil
	// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel()

	// // 构建请求
	// request := &schema.ChatRequest{
	// 	Prompt: g.prompt,
	// 	Tools:  g.tools,
	// }

	// // 调用模型
	// response, err := g.model.Call(ctx, request)
	// if err != nil {
	// 	return current, err
	// }

	// // 解析响应
	// play, err := parseResponse(response)
	// if err != nil {
	// 	return current, err
	// }

	// return play, nil
}
