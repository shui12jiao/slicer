package controller

import (
	"slicer/util"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetUsedMetrics(t *testing.T) {
	config := util.LoadConfig("../.env")
	metrics, err := NewMetrics(config.MonarchThanosURI)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	// 测试获取指标数据
	sliceID := "1-000001"
	duration := time.Hour
	step := time.Minute
	usedMetrics, err := metrics.GetUsedMetrics(sliceID, duration, step)
	require.NoError(t, err)
	require.NotEmpty(t, usedMetrics)
}
