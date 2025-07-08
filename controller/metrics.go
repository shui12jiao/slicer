package controller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// 所用的指标
type UsedMetrics struct {
	UpThroughput   []float64
	DownThroughput []float64
	Latency        []float64
	Availability   []float64
}

// Result 表示一个查询结果，包含其标签和值
type Result struct {
	Labels map[string]string // 标签，用于识别时间序列
	Values []struct {
		Timestamp time.Time
		Value     float64
	}
}

// ValuesAsFloats 是 Result 的一个辅助方法，用于将该时间序列的所有值提取为 []float64。
// 如果 Result 中没有值，它将返回一个空切片。
func (r *Result) ValuesAsFloats() []float64 {
	if r == nil || len(r.Values) == 0 {
		return []float64{}
	}
	floatValues := make([]float64, 0, len(r.Values))
	for _, val := range r.Values {
		floatValues = append(floatValues, val.Value)
	}
	return floatValues
}

// 抽象指标采集接口（兼容内部/外部数据源）
type Metrics interface {
	GetUsedMetrics(sliceID string, duration, step time.Duration) (UsedMetrics, error)  // 获取所用的指标
	GetUpThroughput(sliceID string, duration, step time.Duration) ([]float64, error)   // Mbps
	GetDownThroughput(sliceID string, duration, step time.Duration) ([]float64, error) // Mbps
	GetLatency(sliceID string, duration, step time.Duration) ([]float64, error)        // ms
	GetAvailability(sliceID string, duration, step time.Duration) ([]float64, error)   // %

	QueryRange(query string, start, end time.Time, step time.Duration) ([]Result, error)
}

type ThanosMetrics struct {
	Client v1.API
}

func NewMetrics(uri string) (Metrics, error) {
	cfg := api.Config{
		Address: uri,
	}
	client, err := api.NewClient(cfg)
	if err != nil {
		slog.Error("创建 Thanos 客户端失败", "error", err)
		return nil, err
	}

	return &ThanosMetrics{
		Client: v1.NewAPI(client),
	}, nil
}

func (tm *ThanosMetrics) QueryRange(query string, start, end time.Time, step time.Duration) ([]Result, error) {
	r := v1.Range{
		Start: start.UTC(), // 确保时间是 UTC
		End:   end.UTC(),
		Step:  step,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 整个查询的上下文超时
	defer cancel()

	// QueryRange 返回结果、警告和错误
	// v1.WithTimeout() 是针对 Prometheus 服务器的查询超时，与 context.WithTimeout 不同
	result, warnings, err := tm.Client.QueryRange(ctx, query, r, v1.WithTimeout(20*time.Second))
	if err != nil {
		slog.Error("查询 Thanos 指标失败", "query", query, "range", r, "error", err)
		return nil, fmt.Errorf("prometheus 查询失败: %w", err)
	}
	if len(warnings) > 0 {
		slog.Warn("查询 Thanos 指标时有警告", "query", query, "warnings", warnings)
	}

	// 根据结果类型进行处理
	switch result.Type() {
	case model.ValMatrix: // 最常见的情况，表示多条时间序列，每条包含一系列时间点的值
		matrix, ok := result.(model.Matrix)
		if !ok { // 类型断言失败，理论上不会发生，但为了安全加上
			slog.Error("类型断言失败，期望 model.Matrix", "actual_type", result.Type())
			return nil, fmt.Errorf("内部错误: 期望 model.Matrix 但得到 %s", result.Type())
		}

		if len(matrix) == 0 {
			slog.Info("查询结果为空矩阵", "query", query)
			return []Result{}, nil // 没有匹配的时间序列，返回空切片
		}

		results := make([]Result, 0, len(matrix))
		for _, series := range matrix { // 遍历每个时间序列
			// 将 model.Metric 转换为 map[string]string
			labels := make(map[string]string, len(series.Metric))
			for k, v := range series.Metric {
				labels[string(k)] = string(v)
			}

			// 提取每个时间序列的所有数据点
			values := make([]struct {
				Timestamp time.Time
				Value     float64
			}, 0, len(series.Values))

			for _, sample := range series.Values {
				values = append(values, struct {
					Timestamp time.Time
					Value     float64
				}{
					Timestamp: sample.Timestamp.Time(), // 转换为 Go 的 time.Time
					Value:     float64(sample.Value),
				})
			}

			// 如果某个时间序列在查询范围内没有数据点，`values` 将为空。
			// 根据业务需求决定是否包含到结果中，这里选择包含，但其 Values 字段会是空。
			results = append(results, Result{
				Labels: labels,
				Values: values,
			})
		}
		return results, nil

	case model.ValVector: // 即时向量，表示在查询时间点（或 end 时间点）的多个时间序列的单个值
		vector, ok := result.(model.Vector)
		if !ok {
			slog.Error("类型断言失败，期望 model.Vector", "actual_type", result.Type())
			return nil, fmt.Errorf("内部错误: 期望 model.Vector 但得到 %s", result.Type())
		}
		if len(vector) == 0 {
			slog.Info("查询结果为空向量", "query", query)
			return []Result{}, nil
		}

		results := make([]Result, 0, len(vector))
		for _, sample := range vector {
			labels := make(map[string]string, len(sample.Metric))
			for k, v := range sample.Metric {
				labels[string(k)] = string(v)
			}
			results = append(results, Result{
				Labels: labels,
				Values: []struct {
					Timestamp time.Time
					Value     float64
				}{{Timestamp: sample.Timestamp.Time(), Value: float64(sample.Value)}},
			})
		}
		return results, nil

	case model.ValScalar: // 单个数值，表示一个时间点的一个数值
		scalar, ok := result.(*model.Scalar) // Scalar 是一个指针类型
		if !ok {
			slog.Error("类型断言失败，期望 model.Scalar", "actual_type", result.Type())
			return nil, fmt.Errorf("内部错误: 期望 model.Scalar 但得到 %s", result.Type())
		}
		slog.Info("查询结果为标量", "query", query, "value", float64(scalar.Value))
		return []Result{
			{
				Labels: map[string]string{"__name__": "scalar_result"}, // 标量没有标签，可以给一个默认标签
				Values: []struct {
					Timestamp time.Time
					Value     float64
				}{{Timestamp: scalar.Timestamp.Time(), Value: float64(scalar.Value)}},
			},
		}, nil

	case model.ValString: // 字符串类型结果 (不常见于指标查询)
		strVal, ok := result.(*model.String) // String 也是一个指针类型
		if !ok {
			slog.Error("类型断言失败，期望 model.String", "actual_type", result.Type())
			return nil, fmt.Errorf("内部错误: 期望 model.String 但得到 %s", result.Type())
		}
		slog.Warn("PromQL 查询返回了字符串类型结果，这通常不是指标值", "query", query, "value", strVal.Value)
		return nil, fmt.Errorf("PromQL 查询返回了字符串类型结果: %s", strVal.Value)

	default: // 未知或不支持的结果类型
		slog.Error("查询 Thanos 指标结果类型不支持或未知", "query", query, "type", result.Type())
		return nil, fmt.Errorf("不支持或未知的结果类型: %s", result.Type())
	}
}

// slice_throughput抓取间隔一般为1s
// step表示对metrics进行聚合的时间间隔, 例如1m, 5m, 10m
// avg_over_time的窗口大小这里直接使用step, 也可以使用更大的窗口保证数据的平滑
func (tm *ThanosMetrics) GetUpThroughput(sliceID string, duration, step time.Duration) ([]float64, error) {
	// slice_throughput表示某一时刻吞吐量, 单位Mbps
	query := fmt.Sprintf(`avg_over_time(slice_throughput{direction="uplink", snssai="%s"}[%s])`, sliceID, step)
	end := time.Now()
	start := end.Add(-duration)

	res, err := tm.QueryRange(query, start, end, step)
	if err != nil || len(res) == 0 {
		return nil, err
	}
	// 提取结果中的值
	return res[0].ValuesAsFloats(), nil
}

func (tm *ThanosMetrics) GetDownThroughput(sliceID string, duration, step time.Duration) ([]float64, error) {
	// slice_throughput表示某一时刻吞吐量, 单位Mbps
	query := fmt.Sprintf(`avg_over_time(slice_throughput{direction="downlink", snssai="%s"}[%s])`, sliceID, step)
	end := time.Now()
	start := end.Add(-duration)

	res, err := tm.QueryRange(query, start, end, step)
	if err != nil || len(res) == 0 {
		return nil, err
	}
	// 提取结果中的值
	return res[0].ValuesAsFloats(), nil
}

func (tm *ThanosMetrics) GetLatency(sliceID string, duration, step time.Duration) ([]float64, error) {
	// TODO
	// 暂时没有找到合适的指标可以表示延迟
	return nil, nil
}

func (tm *ThanosMetrics) GetAvailability(sliceID string, duration, step time.Duration) ([]float64, error) {
	// fivegs_smffunction_sm_pdusessioncreationsucc以及fivegs_smffunction_sm_pdusessioncreationfail
	querySucc := fmt.Sprintf(`avg_over_time(fivegs_smffunction_sm_pdusessioncreationsucc{snssai="%s"}[%s])`, sliceID, step)
	queryFail := fmt.Sprintf(`avg_over_time(fivegs_smffunction_sm_pdusessioncreationfail{snssai="%s"}[%s])`, sliceID, step)
	end := time.Now()
	start := end.Add(-duration)

	resSucc, err := tm.QueryRange(querySucc, start, end, step)
	if err != nil || len(resSucc) == 0 {
		return nil, err
	}
	resFail, err := tm.QueryRange(queryFail, start, end, step)
	if err != nil || len(resFail) == 0 {
		return nil, err
	}

	valuesSucc := resSucc[0].ValuesAsFloats()
	valuesFail := resFail[0].ValuesAsFloats()

	availability := make([]float64, len(valuesSucc))
	for i := range valuesSucc {
		if valuesSucc[i] == 0 { // 避免除0错误
			availability[i] = 0
		} else {
			availability[i] = valuesSucc[i] / (valuesSucc[i] + valuesFail[i])
		}
	}
	return availability, nil
}

func (tm *ThanosMetrics) GetUsedMetrics(sliceID string, duration, step time.Duration) (um UsedMetrics, err error) {
	upthroughput, err := tm.GetUpThroughput(sliceID, duration, step)
	if err != nil {
		return
	}

	downthroughput, err := tm.GetDownThroughput(sliceID, duration, step)
	if err != nil {
		return
	}

	latency, err := tm.GetLatency(sliceID, duration, step)
	if err != nil {
		return
	}

	availability, err := tm.GetAvailability(sliceID, duration, step)
	if err != nil {
		return
	}

	um = UsedMetrics{
		UpThroughput:   upthroughput,
		DownThroughput: downthroughput,
		Latency:        latency,
		Availability:   availability,
	}

	return
}
