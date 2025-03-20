package model

import "time"

type Monitor struct {
	APIVersion         string             `json:"api_version" yaml:"api_version"`
	RequestDescription string             `json:"request_description" yaml:"request_description"`
	Scope              Scope              `json:"scope" yaml:"scope"`
	KPI                KPI                `json:"kpi" yaml:"kpi"`
	Duration           Duration           `json:"duration" yaml:"duration"`
	MonitoringInterval MonitoringInterval `json:"monitoring_interval" yaml:"monitoring_interval"`
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
