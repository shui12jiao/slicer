package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"slicer/controller"
	sm "slicer/model"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type MetricsTool struct {
	controller.Metrics
	tool.BaseTool
}

type MetricsToolParams struct {
	SliceID  string `json:"slice_id"`
	Duration int64  `json:"duration"` // in seconds
	Step     int64  `json:"step"`     // in seconds
}

func (m *MetricsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "Metrics Fetcher",
		Desc: "获取切片的指标数据",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"slice_id": {
					Type:     schema.String,
					Desc:     "切片ID",
					Required: true,
				},
				"duration": {
					Type:     schema.Integer,
					Desc:     "持续时间（秒）",
					Required: true,
				},
				"step": {
					Type:     schema.Integer,
					Desc:     "采样间隔（秒）",
					Required: true,
				},
			},
		),
	}, nil
}

func (m *MetricsTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 参数解析
	var params MetricsToolParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", err
	}

	// 获取指标数据
	metrics, err := m.GetUsedMetrics(params.SliceID, time.Duration(params.Duration)*time.Second, time.Duration(params.Step)*time.Second)
	if err != nil {
		return "", err
	}

	// 处理指标数据为JSON格式
	data, err := json.Marshal(metrics)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

type StrategyAgent struct {
	Model       model.ToolCallingChatModel
	MetricsTool tool.InvokableTool
	Agent       *react.Agent
}

func NewStrategyAgent(ctx context.Context, metricsTool tool.InvokableTool, model model.ToolCallingChatModel) (*StrategyAgent, error) {
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{metricsTool},
		},
		MaxStep: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("创建策略代理失败: %w", err)
	}

	return &StrategyAgent{
		Model:       model,
		MetricsTool: metricsTool,
		Agent:       agent,
	}, nil
}

func (s *StrategyAgent) Reconcile(current sm.Play, sla sm.SLA) (sm.Play, error) {
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
