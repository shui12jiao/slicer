package controller

import (
	"fmt"
	"math"
	"slicer/model"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Strategy interface {
	// 返回策略名称
	Name() string
	// 根据SLA,Metrics以及当前策略，生成新 Play 策略
	Reconcile(current model.Play, sla model.SLA) (model.Play, error)
}

// 简易策略
// 该策略根据SLA和当前指标，简单地调整Play策略
// 主要用于测试和验证
type BasicStrategy struct {
	// 指标采集器
	Metrics Metrics
}

func NewBasicStrategy(metrics Metrics) *BasicStrategy {
	return &BasicStrategy{
		Metrics: metrics,
	}
}

func (b *BasicStrategy) Name() string {
	return "basic"
}

func (b *BasicStrategy) Reconcile(current model.Play, sla model.SLA) (model.Play, error) {
	play := &current

	// 获取指标数据（保持原有错误处理）
	metrics, err := b.Metrics.GetUsedMetrics(current.SliceID, 3*time.Hour, time.Minute)
	if err != nil {
		return current, fmt.Errorf("获取指标数据失败: %w", err)
	}

	// 核心调整逻辑（直接使用SLA数值）
	if err := b.adjustBandwidth(play, metrics, sla); err != nil {
		return current, err
	}
	if err := b.adjustAvailability(play, metrics, sla); err != nil {
		return current, err
	}

	return *play, nil
}

func (b *BasicStrategy) adjustBandwidth(play *model.Play, metrics UsedMetrics, sla model.SLA) error {
	// 获取当前配置带宽
	currentUp := parseBandwidthMbps(play.Bandwidth.Ingress)
	currentDown := parseBandwidthMbps(play.Bandwidth.Egress)

	// 计算峰值需求（P95）
	p95Up := calculatePercentile(metrics.UpThroughput, 95)
	p95Down := calculatePercentile(metrics.DownThroughput, 95)

	// 上行带宽调整规则
	switch {
	case p95Up > currentUp*0.9: // 峰值超过当前带宽90%
		newUp := math.Ceil(p95Up * 1.2) // 扩容20%并向上取整
		play.Bandwidth.Ingress = fmt.Sprintf("%.0fM", newUp)
	case p95Up < currentUp*0.5: // 长期低负载
		newUp := math.Max(p95Up*1.1, sla.UpBandwidth) // 最低保持SLA要求
		play.Bandwidth.Ingress = fmt.Sprintf("%.0fM", newUp)
	}

	// 下行带宽调整规则
	switch {
	case p95Down > currentDown*0.9:
		newDown := math.Ceil(p95Down * 1.2)
		play.Bandwidth.Egress = fmt.Sprintf("%.0fM", newDown)
	case p95Down < currentDown*0.5:
		newDown := math.Max(p95Down*1.1, sla.DownBandwidth)
		play.Bandwidth.Egress = fmt.Sprintf("%.0fM", newDown)
	}

	return nil
}

// 带宽解析辅助函数
func parseBandwidthMbps(bw string) float64 {
	if strings.HasSuffix(bw, "M") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(bw, "M"), 64)
		return val
	}
	return 0
}

// 百分位数计算（类型安全的实现）
func calculatePercentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sort.Float64s(data)
	index := (p / 100) * float64(len(data)-1)
	return data[int(math.Ceil(index))]
}

func (b *BasicStrategy) adjustAvailability(play *model.Play, metrics UsedMetrics, sla model.SLA) error {
	// 计算窗口可用性（兼容网页7的SLA计算标准）
	calculateWindowAvailability := func(availabilities []float64) float64 {
		if len(availabilities) == 0 {
			return 100.0
		}
		total := 0.0
		for _, a := range availabilities {
			if a >= 0 && a <= 100 {
				total += a
			}
		}
		return total / float64(len(availabilities))
	}

	if current := calculateWindowAvailability(metrics.Availability); current < sla.Availability {
		// 增强网络策略（参考网页5的网络配置模式）
		if play.NetworkPolicy.Spec.PodSelector.MatchLabels == nil {
			play.NetworkPolicy.Spec.PodSelector.MatchLabels = make(map[string]string)
		}
		play.NetworkPolicy.Spec.PodSelector.MatchLabels["sla-tier"] = "gold"

		// 添加监控标签（参考网页7的SLA监控策略）
		if play.Annotations == nil {
			play.Annotations = make(map[string]string)
		}
		play.Annotations["qos-policy"] = fmt.Sprintf("ha-%.1f", sla.Availability)
		play.Annotations["last-adjusted"] = time.Now().UTC().Format(time.RFC3339) // ISO8601格式[8](@ref)
	}

	return nil
}
