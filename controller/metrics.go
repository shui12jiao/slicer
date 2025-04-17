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

// 抽象指标采集接口（兼容内部/外部数据源）
type Metrics interface {
	GetUsedMetrics(sliceID string, duration, step time.Duration) (UsedMetrics, error)  // 获取所用的指标
	GetUpThroughput(sliceID string, duration, step time.Duration) ([]float64, error)   // Mbps
	GetDownThroughput(sliceID string, duration, step time.Duration) ([]float64, error) // Mbps
	GetLatency(sliceID string, duration, step time.Duration) ([]float64, error)        // ms
	GetAvailability(sliceID string, duration, step time.Duration) ([]float64, error)   // %

	QueryRange(query string, start, end time.Time, step time.Duration) ([]float64, error)
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

func (tm *ThanosMetrics) QueryRange(query string, start, end time.Time, step time.Duration) ([]float64, error) {
	r := v1.Range{
		Start: start.UTC(),
		End:   end.UTC(),
		Step:  step,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, warnings, err := tm.Client.QueryRange(ctx, query, r, v1.WithTimeout(20*time.Second))
	if err != nil {
		slog.Error("查询 Thanos 指标失败", "error", err)
		return nil, err
	}
	if len(warnings) > 0 {
		slog.Warn("查询 Thanos 指标时有警告", "warnings", warnings)
	}

	switch result.Type() {
	case model.ValMatrix:
		matrix := result.(model.Matrix)
		values := make([]float64, len(matrix))
		for i, v := range matrix {
			if len(v.Values) > 0 {
				values[i] = float64(v.Values[0].Value)
			}
		}
		return values, nil
	default:
		slog.Error("查询 Thanos 指标结果类型不支持", "type", result.Type())
		return nil, fmt.Errorf("不支持的结果类型: %s", result.Type())
	}
}

// slice_throughput抓取间隔一般为1s
// step表示对metrics进行聚合的时间间隔, 例如1m, 5m, 10m
// avg_over_time的窗口大小这里直接使用step, 也可以使用更大的窗口保证数据的平滑
func (tm *ThanosMetrics) GetUpThroughput(sliceID string, duration, step time.Duration) ([]float64, error) {
	// slice_throughput表示某一时刻吞吐量, 单位Mbps
	query := fmt.Sprintf(`avg_over_time(slice_throughput{direction="uplink", slice_id="%s"}[%s])`, sliceID, step)
	end := time.Now()
	start := end.Add(-duration)

	return tm.QueryRange(query, start, end, step)
}

func (tm *ThanosMetrics) GetDownThroughput(sliceID string, duration, step time.Duration) ([]float64, error) {
	// slice_throughput表示某一时刻吞吐量, 单位Mbps
	query := fmt.Sprintf(`avg_over_time(slice_throughput{direction="downlink", slice_id="%s"}[%s])`, sliceID, step)
	end := time.Now()
	start := end.Add(-duration)

	return tm.QueryRange(query, start, end, step)
}

func (tm *ThanosMetrics) GetLatency(sliceID string, duration, step time.Duration) ([]float64, error) {
	// TODO
	// 暂时没有找到合适的指标可以表示延迟
	return nil, nil
}

func (tm *ThanosMetrics) GetAvailability(sliceID string, duration, step time.Duration) ([]float64, error) {
	// fivegs_smffunction_sm_pdusessioncreationsucc以及fivegs_smffunction_sm_pdusessioncreationfail
	querySucc := fmt.Sprintf(`avg_over_time(fivegs_smffunction_sm_pdusessioncreationsucc{slice_id="%s"}[%s])`, sliceID, step)
	queryFail := fmt.Sprintf(`avg_over_time(fivegs_smffunction_sm_pdusessioncreationfail{slice_id="%s"}[%s])`, sliceID, step)
	end := time.Now()
	start := end.Add(-duration)

	valuesSucc, err := tm.QueryRange(querySucc, start, end, step)
	if err != nil {
		return nil, err
	}
	valuesFail, err := tm.QueryRange(queryFail, start, end, step)
	if err != nil {
		return nil, err
	}

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
