package model

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SupportedKpi struct {
	KpiName        string `json:"kpi_name"`
	KpiDescription string `json:"kpi_description"`
	KpiUnit        string `json:"kpi_unit"`
	// KpiPromMetricName string `json:"kpi_prom_metric_name"`
}

type Monitor struct {
	ID primitive.ObjectID `json:"id" yaml:"id" bson:"_id,omitempty"`
	// 通用
	APIVersion         string             `json:"api_version" yaml:"api_version"`
	RequestDescription string             `json:"request_description" yaml:"request_description"`
	Scope              Scope              `json:"scope" yaml:"scope"`
	KPI                KPI                `json:"kpi" yaml:"kpi"`
	Duration           Duration           `json:"duration" yaml:"duration"`
	MonitoringInterval MonitoringInterval `json:"monitoring_interval" yaml:"monitoring_interval"`
	// 监控的切片ID
	SliceID string `json:"slice_id" yaml:"slice_id"`
	//用于request translator
	RequestID string `json:"request_id,omitempty" yaml:"request_id,omitempty"`
}

type Scope struct {
	ScopeType string `json:"scope_type" yaml:"scope_type"`
	ScopeID   string `json:"scope_id" yaml:"scope_id"`
}

type KPI struct {
	KPIName        string     `json:"kpi_name" yaml:"kpi_name"`
	KPIDescription string     `json:"kpi_description" yaml:"kpi_description"`
	SubCounter     SubCounter `json:"sub_counter" yaml:"sub_counter"`
	Units          string     `json:"units" yaml:"units"`
}

type SubCounter struct {
	SubCounterType string   `json:"sub_counter_type" yaml:"sub_counter_type"`
	SubCounterIDs  []string `json:"sub_counter_ids" yaml:"sub_counter_ids"`
}

type Duration struct {
	StartTime time.Time `json:"start_time" yaml:"start_time"`
	EndTime   time.Time `json:"end_time" yaml:"end_time"`
}

type MonitoringInterval struct {
	Adaptive     bool `json:"adaptive" yaml:"adaptive"`
	IntervalSecs int  `json:"interval_seconds" yaml:"interval_seconds"`
}

func (m *Monitor) Validate() error {
	// Validate the Monitor struct
	switch m.KPI.KPIName {
	case "slice_throughput":
		if m.Scope.ScopeType == "slice" && m.KPI.SubCounter.SubCounterType == "SNSSAI" && len(m.KPI.SubCounter.SubCounterIDs) == 1 && m.SliceID == m.KPI.SubCounter.SubCounterIDs[0] {
			return nil
		}
		return errors.New("slice_throughput KPI验证失败")
	default:
		return errors.New("不支持的KPI")
	}

}
