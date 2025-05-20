package monitor

import (
	"slicer/util"
	"testing"

	"github.com/stretchr/testify/require"
)

const NODE_IP = "172.18.0.3"

var monitor = NewMonitor(&util.Config{
	MonitorConfig: util.MonitorConfig{
		MonarchThanosURI:            "http://" + NODE_IP + ":31004",
		MonarchRequestTranslatorURI: "http://" + NODE_IP + ":30700",
		MonitorTimeout:              255,
	},
})

func TestGetSupportedKpis(t *testing.T) {
	supportedKpis, err := monitor.GetSupportedKpis()
	require.NoError(t, err)
	require.NotEmpty(t, supportedKpis)
	require.Equal(t, supportedKpis[0].KpiName, "slice_throughput")
}
