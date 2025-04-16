package controller

// 所用的指标
type UsedMetrics struct {
	Bandwidth    float64
	Latency      float64
	Availability float64
}

// 抽象指标采集接口（兼容内部/外部数据源）
type Metrics interface {
	GetBandwidth(sliceID string) (float64, error)    // Mbps
	GetLatency(sliceID string) (float64, error)      // ms
	GetAvailability(sliceID string) (float64, error) // %

	GetMetrics(sliceID string, name string) (float64, error)

	GetUsedMetrics(sliceID string) (UsedMetrics, error) // 获取所用的指标
}

type ThanosMetrics struct {
	URI string
}

func NewMetrics(uri string) Metrics {
	return &ThanosMetrics{
		URI: uri,
	}
}

func (tm *ThanosMetrics) GetUsedMetrics(sliceID string) (um UsedMetrics, err error) {
	bandwidth, err := tm.GetBandwidth(sliceID)
	if err != nil {
		return
	}

	latency, err := tm.GetLatency(sliceID)
	if err != nil {
		return
	}

	availability, err := tm.GetAvailability(sliceID)
	if err != nil {
		return
	}

	um = UsedMetrics{
		Bandwidth:    bandwidth,
		Latency:      latency,
		Availability: availability,
	}

	return
}

func (tm *ThanosMetrics) GetBandwidth(sliceID string) (float64, error) {
	// TODO
	return 0, nil
}

func (tm *ThanosMetrics) GetLatency(sliceID string) (float64, error) {
	// TODO
	return 0, nil
}

func (tm *ThanosMetrics) GetAvailability(sliceID string) (float64, error) {
	// TODO
	return 0, nil
}

func (tm *ThanosMetrics) GetMetrics(sliceID string, name string) (float64, error) {
	// TODO
	return 0, nil
}
