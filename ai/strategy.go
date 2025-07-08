package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slicer/controller"
	sm "slicer/model"
	"strconv"
	"strings"
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
		MetricsTool: metricsTool,
		Agent:       agent,
	}, nil
}

func (s *StrategyAgent) Name() string {
	return "ai"
}

func (s *StrategyAgent) Reconcile(sliceID string, current sm.Deploy, sla sm.SLA) (sm.Deploy, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 获取指标数据
	metricsParams := MetricsToolParams{
		SliceID:  sliceID,
		Duration: time.Hour,
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
		schema.SystemMessage("注意: 当前策略可能为空值, 需要根据SLA和Metrics生成新的策略"),
		schema.SystemMessage("限制: 只需要返回新的策略对应的raw JSON, 不允许任何额外信息或修饰"),
		schema.UserMessage("当前切片SNSSAI: " + sliceID),
		schema.UserMessage("当前Deploy: " + current.String()),
		schema.UserMessage("当前Metrics: " + metrics),
		schema.UserMessage("当前SLA: " + sla.String()),
		schema.UserMessage("请根据以上信息, 生成新的Deploy"),
	}

	// agent处理
	response, err := s.Agent.Generate(ctx, input)
	if err != nil {
		return current, fmt.Errorf("生成响应失败: %w", err)
	}

	// 解析响应
	var deploy sm.Deploy
	jsonStr, err := ExtractJSON(response.Content)
	if err != nil {
		return current, fmt.Errorf("提取JSON失败: %w", err)
	}
	if err := json.Unmarshal([]byte(jsonStr), &deploy); err != nil {
		return current, fmt.Errorf("解析响应失败: %w", err)
	}

	return deploy, nil
}

// ExtractJSON 清洗模型返回字符串中的 JSON 代码块或裸 JSON，返回纯 JSON 字符串
func ExtractJSON(content string) (string, error) {
	content = strings.TrimSpace(content)

	var extractedJSON string

	// 尝试提取 Markdown 代码块 ```(?:json)?\n...\n```
	reCodeBlock := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	if matches := reCodeBlock.FindStringSubmatch(content); len(matches) >= 2 {
		extractedJSON = strings.TrimSpace(matches[1])
		// 尝试解析，如果成功则返回
		if isValidJSON(extractedJSON) {
			return extractedJSON, nil
		}
	}

	// 如果是被双引号包裹的转义 JSON
	// 检查是否以 '"' 开始并以 '"' 结束，且内部可能是一个有效的JSON字符串
	// 简单的startsWith/endsWith判断，然后Unquote
	if len(content) > 1 && strings.HasPrefix(content, `"`) && strings.HasSuffix(content, `"`) {
		unquoted, err := strconv.Unquote(content)
		if err == nil {
			// 再次TrimSpace，因为Unquote可能保留了内部的空白
			unquoted = strings.TrimSpace(unquoted)
			if isValidJSON(unquoted) {
				return unquoted, nil
			}
		}
	}

	// 尝试直接提取裸 JSON（最外层以 { 或 [ 开头、以 } 或 ] 结尾）
	reJSONBoundary := regexp.MustCompile(`(?s)(\{.*\}|\[.*\])`)
	if matches := reJSONBoundary.FindStringSubmatch(content); len(matches) >= 2 {
		extractedJSON = strings.TrimSpace(matches[1])
		if isValidJSON(extractedJSON) {
			return extractedJSON, nil
		}
	}

	return "", errors.New("无法从内容中提取有效 JSON")
}

// isValidJSON 辅助函数，检查字符串是否是有效的 JSON
func isValidJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}
