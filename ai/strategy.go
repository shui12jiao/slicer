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
	SliceID  string        `json:"slice_id"`
	Duration time.Duration `json:"duration"`
	Step     time.Duration `json:"step"`
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
	metrics, err := m.GetUsedMetrics(params.SliceID, params.Duration, params.Step)
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

func (s *StrategyAgent) Name() string {
	return "ai"
}

func (s *StrategyAgent) Reconcile(current sm.Play, sla sm.SLA) (sm.Play, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 获取切片ID
	sliceID := current.SliceID
	if sliceID == "" {
		return current, fmt.Errorf("切片ID不能为空")
	}

	// 获取指标数据
	metricsParams := MetricsToolParams{
		SliceID:  sliceID,
		Duration: 3 * time.Hour,
		Step:     time.Minute,
	}
	metricsParamsJSON, err := json.Marshal(metricsParams)
	if err != nil {
		return current, fmt.Errorf("参数序列化失败: %w", err)
	}
	metrics, err := s.MetricsTool.InvokableRun(ctx, string(metricsParamsJSON))
	if err != nil {
		return current, fmt.Errorf("获取指标数据失败: %w", err)
	}

	// 构建请求
	input := []*schema.Message{
		schema.SystemMessage(StragetyPrompt),
		schema.SystemMessage("限制: 当前策略不能被删除, 只能在当前策略的基础上进行修改, 暂时仅对策略中以下字段进行修改"),
		schema.SystemMessage("1. 资源请求与限制\n2. 带宽限制\n"),
		schema.UserMessage("当前策略: " + current.String()),
		schema.UserMessage("当前指标: " + metrics),
		schema.UserMessage("当前SLA: " + sla.String()),
		schema.UserMessage("请根据当前指标和SLA, 生成新的Play策略"),
		schema.UserMessage("注意: 只需要返回新的策略对应的JSON格式数据, 不要多余描述"),
	}

	// agent处理
	response, err := s.Agent.Generate(ctx, input)
	if err != nil {
		return current, fmt.Errorf("生成响应失败: %w", err)
	}

	// 解析响应
	var play sm.Play
	if err := json.Unmarshal([]byte(response.Content), &play); err != nil {
		return current, fmt.Errorf("解析响应失败: %w", err)
	}

	return play, nil
}
